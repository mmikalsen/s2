package main

import (
    _"fmt"
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
    sCh chan int
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

    c.sCh = make(chan int, 10)
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

            expire, _ := val.(time.Time)
            if expire.After(t1) {
                c.ttl -= c.ttl/16
                //fmt.Printf("\x1b[32;1m■")
                log.Print(string(fetchedKey), ": ok - current ttl: ", c.ttl)
            } else {
                c.ttl += c.ttl/4
                //fmt.Printf("\x1b[0m■")
                log.Print(string(fetchedKey), "ttl failed by:", t1.Sub(expire))
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
    timeout := make(chan []byte, 100)
    go c.TimeoutMonitor(timeout)

    for i := 0; ;i++{
        select {
        case _ = <- c.sCh:
            return
        default:
        }

        key := []byte(hostname + strconv.Itoa(i))
        ttl := time.Now().Add(c.ttl)
        go func() {
            tKey := key
            time.Sleep(c.ttl)
            timeout <- tKey
        }()

        c.index.Insert(key,ttl)
        //log.Print("Sent: ", key, "- expire: ", ttl)
        //fmt.Printf("\x1b[34;1m■")
        c.s.Write(key, remoteAddr)
        time.Sleep(100 * time.Millisecond)
    }
}

func(c *client) TimeoutMonitor(ch chan []byte) {
    for {
        signal := <-ch
        if _, ok := c.index.Lookup(signal); ok {
            //fmt.Printf("\x1b[31;1m■")
            c.ttl = c.ttl + c.ttl/10
            //log.Print("TimeOut")
        }
    }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    c := new(client)
    frontend, err := c.Init(":8090", ":9001")
    if err != nil {
        log.Fatal(err)
    }

    log.Print("start requesting")
    c.Request(frontend)

}
