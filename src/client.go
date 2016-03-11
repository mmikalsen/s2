package main

import (
    "fmt"
    "log"
    "./lib/server"
    "time"
    "os"
    "strconv"
    "net"
    "reflect"
    "runtime"
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
    /*
        0 - tiles channel
        1 - local TTLs  channel
        2 - reciver-requester channel
    */
    chans [4]chan []byte

)



func screenManager(ch chan []byte) {
    var c []byte
    for {
        c = <-ch

        switch string(c) {
        case "B":
            fmt.Printf("\x1b[34;1m■ ")
        case "G":
            fmt.Printf("\x1b[32;1m■ ")
        case "R":
            fmt.Printf("\x1b[31;1m■ ")
        case "W":
            fmt.Printf("\x1b[0m■ ")

        }
    }
}

func timeManager(ch chan []byte) {
    for {
        state := <-ch
        if string(state) == "-" {
            TTL -= 10 * time.Microsecond
        } else {
            TTL += 42 * time.Microsecond
        }
    }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    /* Init logging to file */

    //logfile = new(logS)
    //logfile.Init()

    /* Start Tiles and Timeout-Mananger */
    chans[0] = make(chan []byte, 10)
    chans[1] = make(chan []byte)
    chans[2] = make(chan []byte, 10)

    go screenManager(chans[0])
    /* start UDP server */
    server := new(server.UDPServer)
    server.Init(":9078")

    go reciver(server)


    frontend, err := net.ResolveUDPAddr("udp", "localhost:9001")
    if err != nil {
         log.Fatal(err)
    }


    requester(server, frontend)
}

func requester(server *server.UDPServer, remoteAddr *net.UDPAddr) {

    hostname, _ := os.Hostname()
    for i := 0; ; i++ {
        key := []byte(hostname + strconv.Itoa(i))

        chans[2] <- key
        fmt.Printf("\x1b[34;1m■")
        server.Write(key, remoteAddr)
    }
}

func reciver(server *server.UDPServer) {
     hostname, _ := os.Hostname()
     for i := 0; ; i++ {
         key := []byte(hostname + strconv.Itoa(i))

             select {
                 case requestKey := <- chans[2]:
                    fetchedKey, _, err := server.Read(len(key) + 10)

                 if err != nil {
                      log.Fatal(err)
                 }

                 if reflect.DeepEqual(requestKey, fetchedKey) {
                    fmt.Printf("\x1b[32;1m■")
                } else {
                     fmt.Printf("\x1b[0m■")
                }
                default:
                    time.Sleep(1 * time.Millisecond)
                    i--
         }
         //fmt.Println(string(key))
     }
}
