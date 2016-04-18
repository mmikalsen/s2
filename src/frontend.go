package main

import (
    "net"
    "log"
    "time"
	"strings"
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
    sCh chan string
    ttl time.Duration
}

var (
	PORT = ":9000"
)

func (f *Frontend) Init(load_balancer_addr string) {
    var err error
    f.s = new(server.UDPServer)
    f.s.Init(PORT)
    f.ttl = 2 * time.Second

    f.load_balancer, err = net.ResolveUDPAddr("udp", load_balancer_addr)
    if err != nil {
        log.Fatal(err)
    }

    f.sCh = make(chan string)
    go f.recive()

	// send init message to lb
	f.s.Write([]byte("frontend_up"), f.load_balancer)

	msg := <-f.sCh

	f.backend_addr = msg
	f.s.Write([]byte("ACK"), f.load_balancer)

	f.clients = make([]*net.UDPAddr, 8)
}



func (f *Frontend) recive() {

    for {
        fetchedKey, remoteAddr, err := f.s.Read(32)
        if err != nil {
             log.Fatal(err)
        }
		if remoteAddr.String() == f.load_balancer.String() {
			log.Print("GOT MSG: load_balancer - " + string(fetchedKey))
		} else {
			log.Print("Sending request: ", string(fetchedKey), " to backend")
			go f.httpGet(fetchedKey, remoteAddr)
		}
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
		go f.s.Write(key, addr)
		return
	case <- timeoutCh:
		// timeout
		f.ttl += f.ttl/10
		log.Print("timeout")
	}

}

func (f *Frontend) runtime(debug int) {
	/* LOADBALANCER MSG 
	[status:client:client...]
	*/
	for {
		msg := <-f.sCh
		f.s.Write([]byte("ACK"), f.load_balancer)

		clients := strings.Split(msg, ":")
		status := clients[0]

		if status == "OK" {
			for _, _ = range(clients[1:]) {
				// look up in hashmap //
				// change //
				// reset timer? //
			}
		}

		// print information 
	}
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    frontend := new(Frontend)
    frontend.Init("compute-1-1:9000")
}
