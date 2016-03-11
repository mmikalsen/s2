package main

import (
    "fmt"
    "log"
    "./lib/server"
    "time"
    "os"
    "strconv"
    "net"
    "runtime"
    ctrie "./lib/ctrie"
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
    chans[0] = make(chan []byte, 100)
    chans[1] = make(chan []byte, 100)

    Ctrie := ctrie.New(nil)

    /* start UDP server */
    server := new(server.UDPServer)
    server.Init(":9078")

    go reciver(server, Ctrie)


    frontend, err := net.ResolveUDPAddr("udp", "localhost:9001")
    if err != nil {
         log.Fatal(err)
    }


    requester(server, frontend, Ctrie)
}

func requester(server *server.UDPServer, remoteAddr *net.UDPAddr, ctrie *ctrie.Ctrie) {

    hostname, _ := os.Hostname()
    for i := 0; i < 10000; i++ {
        key := []byte(hostname + strconv.Itoa(i))

        ctrie.Insert(key, time.Now())
        fmt.Printf("\x1b[34;1m■")
        server.Write(key, remoteAddr)

    }
}

func reciver(server *server.UDPServer, ctrie *ctrie.Ctrie) {

     for i := 0; i < 10000; i++ {
        fetchedKey, _, err := server.Read(32)
        if err != nil {
            log.Fatal()
        }
        t1 := time.Now()
        if val, ok := ctrie.Lookup(fetchedKey); ok {

            /* Start a goroutine to remove the key from trie */
            go func() {
                ctrie.Remove(fetchedKey)
            }()

            t0, _ := val.(time.Time)
            if dur := t1.Sub(t0); dur > TTL {
                TTL += TTL/4
                fmt.Printf("\x1b[31;1m■")
            } else {
                TTL -= TTL/8
                fmt.Printf("\x1b[32;1m■")

            }
        } else {
            log.Fatal("NOT FOUND in trie!!!!!")
        }
    }
}
