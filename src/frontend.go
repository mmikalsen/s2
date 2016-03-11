package main

import (
    "fmt"
    "net"
    "log"
    "time"
    "runtime"
    "./lib/server"
)

var (
    LOAD_BALANCER = "localhost:8000"
    TTL = 2 * time.Millisecond
)

func timeManager(ch chan bool) {
    for {
        state := <-ch
        if state {
            TTL -= 10 * time.Microsecond
        } else {
            TTL += 42 * time.Microsecond
        }
    }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    server := new(server.UDPServer)
    server.Init(":9001")
}
