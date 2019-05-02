package models

import (
    "sync"
)

type AliveWorkers struct {
    v map[string]bool
    mux sync.Mutex
}

// Constructor
func NewAliveWorkers() *AliveWorkers {
    s := &AliveWorkers{}
    s.v = make(map[string]bool)
    return s
}

// Updates count for a tenant
func (c *AliveWorkers) Update(key string, status bool) {
    c.mux.Lock()
    // Lock so only one goroutine at a time can access the map c.v.
    c.v[key] = status
    if !status {
        delete(c.v, key)
    }
    c.mux.Unlock()
}

// Returns count of IDs for a tenant
func (c *AliveWorkers) Has(key string) bool {
    c.mux.Lock()
    defer c.mux.Unlock()
    return c.v[key]
}

// Returns worker Map
func (c *AliveWorkers) GetMap() map[string]bool {
    c.mux.Lock()
    defer c.mux.Unlock()
    return c.v
}