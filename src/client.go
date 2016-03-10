package main

import (
    "fmt"
    "net"
    "log"
    //"./lib/logger"
    "time"
)

const (
    PROGRESS = iota
    SUCCESS
    TIMEOUT
    DELAYED_SUCCESS
)

var (
    LOAD_BALANCER = "localhost:8000"
    TTL = 100 * time.Microsecond
)


func screenManager(ch chan int) {
    var c int
    for {
        c = <-ch
        switch c {
        case PROGRESS:
            fmt.Printf("\x1b[34;1m■ ")
        case SUCCESS:
            fmt.Printf("\x1b[32;1m■ ")
        case TIMEOUT:
            fmt.Printf("\x1b[31;1m■ ")
        case DELAYED_SUCCESS:
            fmt.Printf("\x1b[0m■ ")

        }
    }
}

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
    /* Init logging to file */

    //logfile = new(logS)
    //logfile.Init()
    scrCh := make(chan int, 100)
    tManagerCh := make(chan bool, 100)
    go screenManager(scrCh)
    go timeManager(tManagerCh)

    // Connect to load balancer for frontend info

    frontend := "localhost:8099"
    conn, err := net.Dial("tcp", frontend)
    if err != nil {
        log.Fatal(err)
    }


    for {
        scrCh <- PROGRESS
        timeout := make(chan bool, 1)
        recv := make(chan bool, 1)
        go func() {
            time.Sleep(TTL)
            timeout <- true
        }()

        go func() {
            buf := make([]byte, 1024)
            fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
            conn.Read(buf)
            if err != nil {
                log.Fatal(err)
            }
            recv <- true
        }()

        select {
        case <-recv:
            scrCh <- SUCCESS
            tManagerCh <- true
        case <-timeout:
            scrCh <- TIMEOUT
            tManagerCh <- false
        }
        time.Sleep(1 * time.Second)
    }
}
