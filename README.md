# CountApp 📟 🔢

`Distributed counter using Go`

---

### Setup Instructions 📜
- Make sure you have [Go](https://golang.org/doc/install) (v1.10 or later) and [GNU Make](https://www.gnu.org/software/make/) installed.
- Place this folder in your Go Home directory (`$GOPATH`).
- `cd <$GOPATH>/countapp` -> Open the project folder.
- `make` -> This will build and run the application.
- You're all set!
- **Configuration**: Additionally, have a look at `config.yaml`. This file contains the app configurations which can be updated on the fly.
    - *workers*: List of worker hosts(processes).
    - *database*: Database-Folder that contains the tenant wise items. (To ensure true counts and consistency)
    - *worker_logs*: Logs-Folder that contains worker-wise logs.
    - *worker_persist*: Persisit worker aggregations to disk every. (seconds) (To manage load on the database, so relevant tradeoffs can be made).
    - *config_check*: Fetch new config every. (seconds)

---
### Architecture decisions 👾
- Each request that comes to the coordinator looks something like this: `[{"id":"item1", "tenant": "tenant1"}, ..]`
- Since we need to ensure, that each item per tenant is counted only once, we need to store the tenant-wise items somewhere. I decided to implement a simple database (which is just separate items per tenant)
- The architecture chosen tries to achieve **eventual consistency** with fast writes and eventual consistency on reads.
- The coordinator doesn't process the request but it forwards it to one of the *alive workers* randomly which responds back with the response obtained from the worker. 
- If there is a failure in connecting with one of the workers, it retries with another worker. (Ensuring **no loss of requests in case of a network failure or if the worker crashed** for some reason).
- **Coordinator APIs**
    - `POST /items`
        - Forwarded to one of the workers
    - `GET /items/{tenant_id}/count`
        - Forwarded to one of the workers
    - Each request that comes in is pushed to a channel, so it can be concurrently forwarded to one of the workers. `sync.WaitGroup` is also passed along in the channel so correct response(from the correct worker) can be mapped back to the client.
- **Worker APIs**
    - `POST /items`
        - Each item in the request is pushed to a channel so the aggregations can be performed concurrently.
        - Worker maintains a tenant wise `map[string]bool` which is backed up in different files(so can be done concurrently).
        - Worker only persists this to disk(tenant-wise) based on the *worker_persist* config setting.
        - As soon as the worker persists this, it removes it from its own memory and updates the count for the corresponding tenant.
        - See file `models/worker_models.go` for **Safe** implementations of worker specific data structures.
    - `GET /items/{tenant_id}/count`
        - Worker maintains a tenant wise count as well.(Which is not explicitly backed up).
        - This count is updated(fetched from disk) based on the *worker_persist* config setting.
        - See file `models/coordinator_models.go` for **Safe** implementations of coordinator specific data structures.
        - All the database operations(file access) are also made **Safe** using `sync.Mutex`. See `utils/persist.go`
    - `GET /`
        - Health Check for the coordinator to see if the worker is reachable.
    - `POST, GET /kill`
        - For the coordinator to safely kill the worker. The worker persists its aggregations before dying (using `sync.WaitGroup`).
![CountApp Architecture](https://live-wire.github.io/images/countApp.png)

---
### Usage Examples 💻
- Sample usage:
```
countapp $ make
Look at the settings in config.yaml. (You can make changes in the config on the fly ✈️ )
(go run coordinator.go)
Coordinator is up on :5000
🔮  Spawning worker on port:> 5002
🔮  Spawning worker on port:> 5001
🔮  Spawning worker on port:> 5004
🔮  Spawning worker on port:> 5003
POST http://localhost:5003/items
200 OK
POST http://localhost:5004/items
200 OK
GET http://localhost:5001/items/8/count
200 OK {"count":2}
```
- This shows that the requests are forwarded to the workers randomly.
- Removing an item from the config also gracefully kills the worker. 
```
GET http://localhost:5002/items/8/count
200 OK {"count":2}
Killing worker URL:> http://localhost:5004
```
- (Killed)Worker logs:
```
💻 [5004]: Worker up on : 5004
💻 [5004]: POST /items
💻 [5004]: Processing {21 6}
💻 [5004]: Processing {32 7}
💻 [5004]: Processing {41 8}
💻 [5004]: Processing {1 5}
💻 [5004]: db/6 &map[12:true 21:true]
💻 [5004]: db/8 &map[14:true 41:true]
💻 [5004]: db/5 &map[1:true 10:true 11:true 21:true]
💻 [5004]: db/7 &map[13:true 32:true]
💻 [5004]: Gracefully killing worker 5004
```
- Adding a new host to the config spawns a new worker.
```
Killing worker URL:> http://localhost:5004
🔮  Spawning worker on port:> 5005
```

---
### Testing
- Use `make tests` to run all the tests. 
- Wrote a few integration tests to test the sanity of the system. See `test/dummy.go`
- TODO: Unit tests.
- Sample usage

```
------
countapp $ make tests


--- Running unit tests ---
(cd worker; go test -v; cd ..)
=== RUN   TestWorkerHealthCheckHandler
--- PASS: TestWorkerHealthCheckHandler (0.00s)
PASS
ok      countapp/worker 0.012s


--- Running sanity tests ---
(rm -rf db/)
(go run test/dummy.go)
Testing Eventual Consistency:
Coordinator is up on :5000
🔮  Spawning worker on port:> 5001
🔮  Spawning worker on port:> 5003
🔮  Spawning worker on port:> 5004
🔮  Spawning worker on port:> 5002
POST [{"id":"1", "tenant": "t1"},
      {"id":"2", "tenant": "t2"},
      {"id":"2", "tenant": "t3"},
      {"id":"4", "tenant": "t1"}]
POST http://localhost:5004/items
200 OK
GET http://localhost:5002/items/t1/count
200 OK {"count":0}

GET http://localhost:5002/items/t1/count
200 OK {"count":0}

GET http://localhost:5004/items/t1/count
200 OK {"count":2}

Consistency Achieved!
signal: killed

------
```

---
### Scopes of improvement 🐙
- Instead of worker urls in the configuration, a number could be configured. Based on that, a number of workers could be spawned.
- For storage, JSON is currently used for easy readability. This can be easily switched to a compressed representation by overriding the `Encode` and `Decode` functions in `utils/persist.go`.
- Currently, the count per tenant is updated on each node only if a request reaches that node. There could be a routine to keep it up to date. (This should be always better than sudden spikes in Database access when there is too much traffic)
- Keep-alive of the workers is checked only when the config is refreshed. There could be a separate configuration option for this to check it more frequently.
- When the coordinator starts the processes, it starts all the processes at once, this would lead to all the processes trying to persist to the database at the same time. There could be a way to spawn instances spread evenly over the `worker_persist` interval, which would ensure that the load (on the database) is evenly distributed along the time.
- Containerizing this application should also be considered (for quick deployment and predictable behaviour across platforms) as go emits executables which can be easily packaged into very small containers.
- I've also read about gRPC but never used it. That should reduce the response times as compared to HTTP requests(atleast for connections between coordinator and workers).


---
### References
- [Go Tour](https://tour.golang.org/welcome/1) Amazing GoLang resource. 🏆
- [Safe Disk Operations](https://medium.com/@matryer/golang-advent-calendar-day-eleven-persisting-go-objects-to-disk-7caf1ee3d11d) Persisting Go objects to disk.
- External Packages used:
    - `"gopkg.in/yaml.v2"` - For yaml configuration parsing.
    - `"github.com/gorilla/mux"` - For request routing.