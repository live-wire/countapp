package main

import (
  "bytes"
  . "countapp/models"
  "countapp/utils"
  "errors"
  "fmt"
  "github.com/gorilla/mux"
  "io/ioutil"
  "log"
  "math/rand"
  "net/http"
  "os"
  "os/exec"
  "path/filepath"
  "strings"
  "sync"
  "time"
)

type CoordinatorReq struct {
  W http.ResponseWriter
  R *http.Request
  Wg *sync.WaitGroup
}

var reqsToProcess chan CoordinatorReq
var aliveWorkers AliveWorkers

func main() {
  aliveWorkers = *NewAliveWorkers()
  reqsToProcess = make(chan CoordinatorReq)
  go processReqsFromQueue()
  config:= utils.Config()
  os.MkdirAll(config.WorkerLogs, os.ModePerm)
  configTicker := time.NewTicker(time.Duration(int(config.ConfigCheck))* time.Second)
  ZooKeep()
  go func() {
      for _ = range configTicker.C {
          ZooKeep()
      }
  }()

  r := mux.NewRouter()
  r.Host("localhost")
  i := r.PathPrefix("/items").Subrouter()
  i.HandleFunc("", AddToQueue).Methods("POST")
  i.HandleFunc("/{tenant}/count", AddToQueue).Methods("GET")
  fmt.Println("Coordinator is up on :5000")
  log.Fatal(http.ListenAndServe(":5000", r))
}

// Manage workers concurrently by checking the latest config
func ZooKeep() {
  config:= utils.Config()
  configWorkers := make(map[string]bool)
  for _, worker := range config.Workers {
    go StartWorker(worker)
    configWorkers[worker] = true
  }
  for k, _ := range aliveWorkers.GetMap() {
    if _, ok := configWorkers[k]; !ok {
      go StopWorker(k)
      aliveWorkers.Update(k, false)
    }
  }
}

// Kills a worker by making a GET call to /kill
func StopWorker(worker string) {
    url := worker
    fmt.Println("Killing worker URL:>", url)
    req, err := http.NewRequest("GET", worker+"/kill", bytes.NewBuffer([]byte("")))
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        //panic(err)
    }
    defer resp.Body.Close()
    //fmt.Println("response Status:", resp.Status)
}

// Health checks a worker or starts it
func StartWorker(worker string) {
    config:= utils.Config()
    url := worker
    req, err := http.NewRequest("GET", worker+"/", bytes.NewBuffer([]byte("")))
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        //panic(err)
        aliveWorkers.Update(worker, false)
    } else {
      defer resp.Body.Close()
      if resp.StatusCode == 200 {
        aliveWorkers.Update(worker, true)
        return
      }
    }
    port:= url[strings.LastIndex(url, ":")+1: ]
    logFilePath := filepath.Join(config.WorkerLogs, port)
    fmt.Println("🔮  Spawning worker on port:>", port)
    cmd := exec.Command("go", "run", "worker/worker.go", port)

    outfile, err := os.Create(logFilePath)
    if err != nil {
        panic(err)
    }
    defer outfile.Close()
    cmd.Stdout = outfile
    err = cmd.Start(); if err != nil {
        panic(err)
    }
    cmd.Wait()
}

// Get one of the Alive hosts randomly
func GetAliveHostRandomly() string {
  workers := aliveWorkers.GetMap()
  if len(workers) < 1 {
    return ""
  }
  i := rand.Intn(len(workers))
  var k string
  for k = range workers {
    if i == 0 {
      break
    }
    i--
  }
  return k
}

// Processes requests in the channel concurrently
func processReqsFromQueue() {
    for {
        item := <- reqsToProcess
        go func (it CoordinatorReq) {
          err := Forward(it.W, it.R, it.Wg)
          if err!= nil {
            reqsToProcess <- it
          }
        } (item)
    }
}

// Adds the incoming request to a channel concurrently
func AddToQueue(w http.ResponseWriter, r *http.Request) {
  var wg sync.WaitGroup
  wg.Add(1)
  req := CoordinatorReq{w, r, &wg}
  go func() {reqsToProcess<- req}()
  wg.Wait()
}

// Forwards the request to one of the workers and returns the response
func Forward(w http.ResponseWriter, r *http.Request, wg *sync.WaitGroup) error {
    url := GetAliveHostRandomly()
    if url == "" {
      return errors.New("No workers are ready")
    }
    fmt.Println(r.Method, url+r.RequestURI)

    b, err := ioutil.ReadAll(r.Body)
    defer r.Body.Close()
    if err != nil {
      //http.Error(w, err.Error(), 500)
      fmt.Println("Failed while reading body", err)
      wg.Done()
      return nil
    }

    var jsonStr = []byte(b)
    req, err := http.NewRequest(r.Method, url+r.RequestURI, bytes.NewBuffer(jsonStr))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("RETRYING: Error while sending request -", err)
        ZooKeep()
        return err
    }
    defer resp.Body.Close()
    w.Header().Set("Content-Type", "application/json")
    body, _ := ioutil.ReadAll(resp.Body)
    w.WriteHeader(resp.StatusCode)
    w.Write(body)
    fmt.Println(resp.Status, string(body))
    wg.Done()
    return nil
}

