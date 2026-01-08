package main

import (
    "log"
    "net/http"

    "api.dauntless-society.com/anchor/handlers"
)

func main() {
    http.HandleFunc("/api/v1/anchor", handlers.AnchorHandler)
    log.Println("Anchor API listening on :8080")
    log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
