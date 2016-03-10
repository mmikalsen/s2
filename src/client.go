package main

import (
    "fmt"
    "math/rand"
    "./shared/logger"
)

const (
    PROGRESS = iota
    SUCCESS
    TIMEOUT
    DELAYED_SUCCESS
)



func screenManager(ch chan int) {
    var c int
    for {
        c = <-ch
        switch c {
        case PROGRESS:
            fmt.Printf("\x1b[34;1m■")
        case SUCCESS:
            fmt.Printf("\x1b[32;1m■")
        case TIMEOUT:
            fmt.Printf("\x1b[31;1m■")
        case DELAYED_SUCCESS:
            fmt.Printf("\x1b[0m■")

        }
    }
}


func main() {
    /* Init logging to file */

    logfile = new(logS)
    logfile.Init()
    scrCh := make(chan int)
    go screenManager(scrCh)
    //go timeManager()

    for j := 0; j < 10; j++{
        i := rand.Int() % 4
        scrCh <- i
    }
    log.Fatal()
}
