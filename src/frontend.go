package main

import (
    "net"
    "log"
    "time"
	"strings"
    "runtime"
    "./lib/server"
    "./lib/config"
    "net/http"
    "github.com/streamrail/concurrent-map"
)

type HttpResponse struct {
	resp *http.Response
	err error
}

type Frontend struct {
    s *server.UDPServer
    load_balancer *net.UDPAddr
    backend_addr string
    clients cmap.ConcurrentMap
    sCh chan string
    ttl time.Duration
}

var (
    conf = new(config.Configuration)
)

func (f *Frontend) Init() {
    var err error
    f.s = new(server.UDPServer)
    f.s.Init(conf.FrontendPort)
    f.ttl = time.Duration(conf.FrontendInitTTL) * time.Millisecond

    f.load_balancer, err = net.ResolveUDPAddr("udp", conf.LB[0] + conf.LBPort)
    if err != nil {
        log.Fatal(err)
    }

    f.sCh = make(chan string)
    go f.recive()

  log.Println(f.load_balancer.String())

	// send init message to lb
	f.s.Write([]byte("frontend_up"), f.load_balancer) //TODO: invalid memory address

	msg := <-f.sCh

	f.backend_addr = msg

	f.s.Write([]byte("ACK"), f.load_balancer)

	f.clients = cmap.New()
}



func (f *Frontend) recive() {

    for {
        body, remoteAddr, err := f.s.Read(32) //TODO: Some runtime error here
        if err != nil {
             log.Fatal(err)
        }
		if remoteAddr.String() == f.load_balancer.String() {
      // msg: "clientAddr lease"
      log.Print("GOT MSG: load_balancer - " + string(body))
      if (f.clients.Count() >= MAXCLIENTS) {
        log.Print("Client limit reached. Aborting receive")
        return
      }
      msg := strings.Split(string(body), " ")

      // The lease is multiple "words" in the string, so join them together again
      lease, err  := time.Parse(time.Stamp, strings.Join(msg[1:], " "))
			if err != nil {
				log.Fatal(err)
			}
      clientAddr := msg[0]
      f.clients.Set(clientAddr, lease)

      go func() {
        // Remove the client once the lease runs out
				time.Sleep(lease.Sub(time.Now()))
				f.clients.Remove(clientAddr)
			}()
		} else {
      _, ok := f.clients.Get(remoteAddr.String())
      if !ok {
        log.Print("Lease has run out.. Aborting")
        return
			}

			log.Print("Sending request: ", string(body), " to backend")
			go f.httpGet(body, remoteAddr)
		}
    }
}

func (f *Frontend) httpGet(key []byte, addr *net.UDPAddr) {
  /*
    Frontend --GET--> Backend
    */
	ttl := f.ttl
	timeoutCh := make(chan bool)
	responseCh := make(chan http.Response)

	go func() {
		time.Sleep(ttl)
		timeoutCh <- true
	}()
	go func() {
		resp, err := http.Get("http://" + f.backend_addr)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		responseCh <- *resp
	}()

	select {
	case r := <-responseCh:
		// got response before timeout
		if r.StatusCode != 200 {
			return
		}
		f.s.Write(key, addr)
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
      for _ = range f.clients.Iter() {
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

    err := conf.GetConfig("config.json")
    if err != nil {
        log.Fatal(err)
    }

	log.SetFlags(log.LstdFlags | log.Lshortfile)

    frontend := new(Frontend)
    frontend.Init()
}
