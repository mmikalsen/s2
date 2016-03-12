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


type client struct {
    s *server.UDPServer
    load_balancer *net.UDPAddr
    frontend *net.UDPAddr
    sCh chan bool
    ttl time.Duration
    index *ctrie.Ctrie
}

func (c *client) Init(port string, load_balancer_addr string) (*net.UDPAddr, error) {
    var err error

    c.s = new(server.UDPServer)
    c.s.Init(port)

    c.load_balancer, err = net.ResolveUDPAddr("udp", load_balancer_addr)
    if err != nil {
        return nil, err
    }

    // index data structure
    c.index = ctrie.New(nil)

    // start listing to udp stream
    go c.recive()

    // fetch frondtend from load_balancer
    c.frontend, err = net.ResolveUDPAddr("udp", ":9001")
    if err != nil {
         // shutdown socket
        return nil, err
     }

    // chans

    c.ttl = 10 * time.Millisecond
    return c.frontend, nil
}

func (c *client) recive() {

    for {
        fetchedKey, remoteAddr, err := c.s.Read(32)
        if err != nil {
            log.Fatal()
        }
        t1 := time.Now()
        if val, ok := c.index.Lookup(fetchedKey); ok {

            /* Start a goroutine to remove the key from trie */
            go func() {
                c.index.Remove(fetchedKey)
            }()

            t0, _ := val.(time.Time)
            if dur := t1.Sub(t0); dur > c.ttl {
                c.ttl += c.ttl/4
                fmt.Printf("\x1b[31;1m■")
            } else {
                c.ttl -= c.ttl/16
                fmt.Printf("\x1b[32;1m■")

            }
        } else {
            /* Control signal */
            if c.frontend == remoteAddr  {

            } else if c.load_balancer == remoteAddr {

            }

        }
    }
}

func (c *client) Request(remoteAddr *net.UDPAddr) {
    hostname, _ := os.Hostname()
    for i := 0; i < 10000; i++ {
        key := []byte(hostname + strconv.Itoa(i))

        c.index.Insert(key, time.Now())
        fmt.Printf("\x1b[34;1m■")
        c.s.Write(key, remoteAddr)

    }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    c := new(client)
    frontend, err := c.Init(":8090", ":9001")
    if err != nil {
        log.Fatal(err)
    }

    c.Request(frontend)

    time.Sleep(10 * time.Second)
}
