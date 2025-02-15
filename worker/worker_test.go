package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestWorkerHealthCheckHandler(t *testing.T) {
    req, err := http.NewRequest("GET", "/", nil)
    if err != nil {
        t.Fatal(err)
    }
    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(HealthCheck)
    handler.ServeHTTP(rr, req)

    // Check the status code is 200
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusOK)
    }
}
