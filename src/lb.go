package main

import  (
  "log"
  "net"
  "./lib/server"
  _"strings"
  "strconv"
  "sync/atomic"
  "runtime"
  "time"
  "github.com/streamrail/concurrent-map"
  "errors"
)

var (
	LEASETIME = 10 * time.Second
	MAXCLIENTS = int32(2)
)

type route struct {
	client *net.UDPAddr
	frontend *net.UDPAddr
	lease time.Time
}

type lbFrontend struct {
	addr *net.UDPAddr
	numClients *int32
	up *int32
}

type lb struct {
	routes cmap.ConcurrentMap
	frontends [3]lbFrontend
	s *server.UDPServer
}

func(l *lb) Init(port string) error {
	l.routes = cmap.New()

	l.s = new(server.UDPServer)
	err := l.s.Init(port)
	if err !=nil {
		return err
	}


	// Make referance to frontend servers \\
	for i := range(l.frontends) {
		f := &l.frontends[i]
		f.addr, err = net.ResolveUDPAddr("udp", "compute-10-" + strconv.Itoa((i + 1)) +":9001")
		if err != nil {
			return err
		}
		f.numClients = new(int32); atomic.StoreInt32(f.numClients, 0)
		f.up = new(int32); atomic.StoreInt32(f.up, 0)
	}
	return nil
}

func (l *lb) Serve() {
	for {
		msg, remoteAddr, err := l.s.Read(32)
		if err != nil {
			log.Fatal(err)
		}
		if string(msg) == "new_lease" {
			frontend, lease, err := l.NewClient(remoteAddr)
			if err != nil {
				log.Fatal(err)
			}
			l.s.Write([]byte(frontend.addr.String() + " " + lease.Format(time.UnixDate)), remoteAddr)
		}
	}
}

func (l *lb) NewClient(client *net.UDPAddr) (*lbFrontend, time.Time, error) {

	for i := range(l.frontends) {
		frontend := &l.frontends[i]
		if atomic.LoadInt32(frontend.up) == 1 && atomic.LoadInt32(frontend.numClients) < MAXCLIENTS {
			route := &route{client, frontend.addr, time.Now().Add(LEASETIME)}
			// Timer
			go func() {
				time.Sleep(route.lease.Sub(time.Now()))
				l.routes.Remove(client.String())
				atomic.AddInt32(frontend.numClients, -1)
			}()
			l.routes.Set(client.String(), route)
			atomic.AddInt32(frontend.numClients, 1)
			return frontend, route.lease, nil
		}
	}

	// need to start another frontend!
	for i := range(l.frontends) {
		frontend := &l.frontends[i]
		if atomic.LoadInt32(frontend.up) != 1 {
			// Start up the frontend by script or just let them sit idle?
			atomic.StoreInt32(frontend.up, 1)
			return l.NewClient(client)
		}
	}
	return nil, time.Now(), errors.New("Frontend is not up!")
}


func (l *lb) Info() {
	for {
		time.Sleep(5 * time.Second)
		for i := range(l.frontends) {
			// mainteines aka turn off not used frontends... if slots left < MAX - 2 or something

			log.Printf("%s has %d clients\n", l.frontends[i].addr.String(), atomic.LoadInt32(l.frontends[i].numClients))
			// propegate information to frontends?
		}
	}
}


func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lb := new(lb)
	err := lb.Init(":9000")
	if err != nil {
		log.Fatal(err)
	}
	go lb.Info()
	lb.Serve()
}
