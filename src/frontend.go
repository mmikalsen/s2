package main

import (
    "net"
    "log"
    "time"
    "runtime"
    "./lib/server"
    "net/http"
)

type HttpResponse struct {
	resp *http.Response
	err error
}

type Frontend struct {
    s *server.UDPServer
    load_balancer *net.UDPAddr
    backend_addr string
    clients []*net.UDPAddr
    sCh chan int
    ttl time.Duration
}

func (f *Frontend) Init(port string, load_balancer_addr string) {
    var err error
    f.s = new(server.UDPServer)
    f.s.Init(port)
    f.ttl = 2 * time.Second

    f.load_balancer, err = net.ResolveUDPAddr("udp", load_balancer_addr)
    if err != nil {
        log.Fatal(err)
    }

    f.sCh = make(chan int, 10)
    go f.recive()

    // make itself availble for clients and get backend_addr from load_balancer
    f.backend_addr = "http://compute-10-2:8000"
    f.clients = make([]*net.UDPAddr, 10)
    f.clients[0], err = net.ResolveUDPAddr("udp", ":8090")
    if err != nil {
        log.Fatal(err)
    }
}



func (f *Frontend) recive() {

    for {
        fetchedKey, remoteAddr, err := f.s.Read(32)
        if err != nil {
             log.Fatal(err)
        }
		log.Print("Sending request: ", string(fetchedKey), " to backend")
        go f.httpGet(fetchedKey, remoteAddr)
    }
}

func (f *Frontend) httpGet(key []byte, addr *net.UDPAddr) {
	ttl := f.ttl
	timeoutCh := make(chan bool)
	responseCh := make(chan *HttpResponse)

	go func() {
		time.Sleep(ttl)
		timeoutCh <- true
	}()
	go func() {
		resp, err := http.Get(f.backend_addr)
		responseCh <- &HttpResponse{resp, err}
	}()

	select {
	case r := <-responseCh:
		// got response before timeout
		if r.err != nil && r.resp.StatusCode != 200 {
			return
		}
		f.s.Write(key, addr)
	case <- timeoutCh:
		// timeout
		f.ttl += f.ttl/10
		log.Print("timeout")
	}

}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    frontend := new(Frontend)
    frontend.Init(":9001", ":9001")
    time.Sleep(100 * time.Second)
}
