package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/exec"
    "syscall"
    "time"
)

type CountResponse struct {
    Count int `json:"count"`
}
func SendItems(payload string) {
    var reqString string
    if payload == "" {
        reqString = `[{"id":"1", "tenant": "t1"},
                      {"id":"2", "tenant": "t2"},
                      {"id":"2", "tenant": "t3"},
                      {"id":"4", "tenant": "t1"}]`
    } else {
        reqString = payload
    }
    fmt.Println("POST", reqString)
    var jsonStr = []byte(reqString)
    req, err := http.NewRequest("POST", "http://localhost:5000/items", bytes.NewBuffer(jsonStr))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error while sending request -", err)
    }
    defer resp.Body.Close()
    //w.Header().Set("Content-Type", "application/json")
    //body, err := ioutil.ReadAll(resp.Body)
}

func GetItemsCount(tenant string) string {
    var tenant_id string
    if tenant == "" {
        tenant_id = "1"
    } else {
        tenant_id = tenant
    }
    req, err := http.NewRequest("GET", "http://localhost:5000/items/"+tenant_id+"/count", nil)
    req.Header.Set("Content-Type", "application/json")

    // fmt.Println("GET", tenant_id)
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error while sending request -", err)
    }
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    return string(body)
}

func StartCoordinator() {
    cmd := exec.Command("go", "run", "coordinator.go")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    time.AfterFunc(30*time.Second, func() {
        syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
    })

    err := cmd.Run()
    if err != nil {
        log.Fatalf("cmd.Run() failed with %s\n", err)
    }
}

func main() {
    fmt.Println("Testing Eventual Consistency:")
    go StartCoordinator()
    time.Sleep(time.Duration(5) * time.Second)
    SendItems("")
    for {
        s := GetItemsCount("t1")
        cnt := &CountResponse{}
        err := json.Unmarshal([]byte(s), cnt)
        if err != nil {
            fmt.Println(err)
        }
        if cnt.Count == 2{
            fmt.Println("Consistency Achieved!")
            break
        } else {
            time.Sleep(time.Duration(1)*time.Second)
        }
    }
}