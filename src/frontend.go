package main

import (
    "fmt"
    "net"
    "log"
    "time"
    //"net/http"
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
    tManagerCh := make(chan bool, 100)
    go timeManager(tManagerCh)

    ln, err := net.Listen("tcp", "localhost:8099")
    if err != nil {
        log.Fatal(err)
    }

    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Fatal(err)
        }
        go handleConnection(conn, tManagerCh)
    }
}

func handleConnection(conn net.Conn, tCh chan bool) {
    defer conn.Close()

    for {
        buf := make([]byte, 1024)
        fmt.Println("recv")
        _, err := conn.Read(buf)
        if err != nil {
            log.Fatal()
        }
        conn.Write([]byte("OK"))
    }
}

