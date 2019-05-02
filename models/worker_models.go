package models

import (
    "sync"
)

type Item struct {
    Id string `json:"id"`
    Tenant string `json:"tenant"`
}

type Tenant struct {
    Id string `json:"id"`
    Items map[string]bool `json:"items"`
}

type Response struct {
    Count int `json:"count"`
}

type WorkerTenants struct {
    v   map[string]Tenant
    mux sync.Mutex
}

type WorkerTenantCounts struct {
    v map[string]int
    mux sync.Mutex
}


// Constructor
func NewWorkerTenants() *WorkerTenants {
    s := &WorkerTenants{}
    s.v = make(map[string]Tenant)
    return s
}

// Constructor
func NewWorkerTenantCounts() *WorkerTenantCounts {
    s := &WorkerTenantCounts{}
    s.v = make(map[string]int)
    return s
}

// Adds an item to the corresponding tenant
func (c *WorkerTenants) Add(item Item) {
    c.mux.Lock()
    _, ok := c.v[item.Tenant]
    if !ok {
        s := make(map[string]bool)
        s[item.Id] = true
        c.v[item.Tenant] = Tenant{ Id: item.Tenant, Items: s }
    } else {
        c.v[item.Tenant].Items[item.Id] = true
    }
    c.mux.Unlock()
}


// Adds an item to the corresponding tenant
func (c *WorkerTenants) ForAll(fn func(string, sync.WaitGroup), wg sync.WaitGroup) {
    c.mux.Lock()
    wg.Add(len(c.v))
    for k,_ := range c.v {
        go fn(k, wg)
    }
    c.mux.Unlock()
}

// Fetches the items for a tenant
func (c *WorkerTenants) GetAndDelete(key string) map[string]bool {
    c.mux.Lock()
    defer c.mux.Unlock()
    tenant, ok := c.v[key]
    if !ok {
        return make(map[string]bool)
    }
    delete(c.v, key)
    return tenant.Items
}

// Updates count for a tenant
func (c *WorkerTenantCounts) Update(key string, count int) {
    c.mux.Lock()
    // Lock so only one goroutine at a time can access the map c.v.
    c.v[key] = count
    c.mux.Unlock()
}

// Returns count of IDs for a tenant
func (c *WorkerTenantCounts) Value(key string) int {
    c.mux.Lock()
    defer c.mux.Unlock()
    return c.v[key]
}