package main

import (
    . "countapp/models"
    "countapp/utils"
    "encoding/json"
    "fmt"
    "github.com/gorilla/mux"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"
)

const VERBOSE_LOGS bool = true
var workerPort string
var workerTenants WorkerTenants
var workerTenantCounts WorkerTenantCounts
var processItems chan Item

func main() {
    initialize()
    r := mux.NewRouter()
    r.Host("localhost")
    r.HandleFunc("/", HealthCheck).Methods("GET")
    r.HandleFunc("/kill", KillWorker).Methods("GET", "POST")
    
    i := r.PathPrefix("/items").Subrouter()
    i.HandleFunc("", AddItems).Methods("POST")
    i.HandleFunc("/{tenant}/count", GetCount).Methods("GET")
    
    printer("Worker up on :", workerPort)
    log.Fatal(http.ListenAndServe(":" + workerPort, r))
}

func initialize() {
    argsWithoutProg := os.Args[1:]
    processItems = make(chan Item)
    config := utils.Config()
    backupTicker := time.NewTicker(time.Duration(int(config.WorkerPersist))* time.Second)
    go func() {
        for _ = range backupTicker.C {
            takeBackupRoutine()
        }
    }()
    os.MkdirAll(config.Database, os.ModePerm)
    workerTenants = *NewWorkerTenants()
    workerTenantCounts = *NewWorkerTenantCounts()
    go processItemsFromQueue()
    go takeBackupRoutine()
    if len(argsWithoutProg) < 1 {
        fmt.Println("Provide the port to run on as an argument!")
        os.Exit(0)
    }
    workerPort = argsWithoutProg[0]
}

func printer(a ...interface{}) (n int, err error) {
    s := "💻 ["+workerPort+"]:"
    all := append([]interface{}{s}, a...)
    if VERBOSE_LOGS {
        return fmt.Println(all...)
    } else {
        return 0, nil
    }
}

// GET Health check handler
func HealthCheck(w http.ResponseWriter, r *http.Request) {
    // fmt.Println("HealthCheck")
}

// Safe Kill self handler
func KillWorker(w http.ResponseWriter, r *http.Request) {
    printer("Gracefully killing worker", workerPort)
    // blocking
    takeBackupRoutine()
    go func (){
        time.Sleep(time.Duration(1) * time.Second)
        os.Exit(0)
    }()
}

// Process items concurrently
func processItemsFromQueue() {
    for {
        item := <- processItems
        printer("Processing", item)
        workerTenants.Add(item)
    }
}

// Take backup concurrently
func takeBackupRoutine() {
    // fmt.Println("Taking Backup")
    var wg sync.WaitGroup
    workerTenants.ForAll(takeBackup, wg)
    wg.Wait()
}

// DB Persist
func takeBackup(tenant string, wg sync.WaitGroup) {
    defer wg.Done()
    config := utils.Config()
    filepath := filepath.Join(config.Database, tenant)
    items := workerTenants.GetAndDelete(tenant)
    itemsInDb := make(map[string]bool)
    var mergeFunc = func (dbItems *map[string]bool) {
        for k, v := range items {
            (*dbItems)[k] = v
        }
        workerTenantCounts.Update(tenant, len(*dbItems))
        printer(filepath, dbItems)
    }
    if _, err := os.Stat(filepath); !os.IsNotExist(err) {
        utils.LoadMergeSaveAtomic(filepath, &itemsInDb, mergeFunc)
    } else {
        mergeFunc(&itemsInDb)
        utils.Save(filepath, itemsInDb)
    }
}

// POST Items Handler
func AddItems(w http.ResponseWriter, r *http.Request) {
    decoder := json.NewDecoder(r.Body)
    items := make([]Item,0)
    err := decoder.Decode(&items)
    if err != nil {
        http.Error(w, err.Error(), 500)
        printer(err)
    }
    printer(r.Method, r.RequestURI)
    for _, item := range items {
        if item.Id == "" || item.Tenant == "" {
            printer("Each item should have (id) and (tenant) attributes", item)
        } else {
            // non blocking addition to queue
            go func (itemLocal Item){processItems <- itemLocal}(item)
        }
    }
}

// GET Count Handler
func GetCount(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    tenant_id := vars["tenant"]
    config := utils.Config()
    // config := utils.Config()
    printer(r.Method, r.RequestURI)
    count := workerTenantCounts.Value(tenant_id)
    if count == 0 {
        printer("Fetching count from db")
        filepath := filepath.Join(config.Database, tenant_id)
        itemsInDb := make(map[string]bool)
        if _, err := os.Stat(filepath); !os.IsNotExist(err) {
            utils.Load(filepath, &itemsInDb)
        }
        count = len(itemsInDb)
        workerTenantCounts.Update(tenant_id, count)
    } else {
        go func() {
            time.Sleep(time.Duration(config.WorkerPersist)*time.Second)
            workerTenantCounts.Update(tenant_id, 0)
        }()
    }
    resp := &Response{count}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
