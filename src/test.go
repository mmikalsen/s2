package main

import (
    "fmt"
    "log"
    "./lib/server"
    "time"
    "math/rand"
)

func main() {
    rand.Seed(time.Now().Unix())
    s := new(server.UDPServer)
    s.Init(":9001")

    for i := 0; i < 10000; i++ {
        dat, remoteAddr, err := s.Read(32)
        fmt.Println(string(dat))
        if err != nil {
            log.Fatal(err)
        }

        go func() {
            time.Sleep( time.Duration(rand.Int31n(10))* time.Millisecond)
            s.Write(dat, remoteAddr)

        }()
    }
   time.Sleep(5 * time.Second)
   s.Conn.Close()
}
