package main

import  (
  "log"
  "net"
  "./lib/server"
  "./lib/config"
  "./lib/logger"
  "sync/atomic"
  "runtime"
  "time"
  "github.com/streamrail/concurrent-map"
  "errors"
  "os"
  "fmt"
  "flag"
)

var (
	conf = new(config.Configuration)
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
	backend string
	sCh chan string
	log *log.Logger
}

func(l *lb) Init() error {
	l.routes = cmap.New()

	l.s = new(server.UDPServer)
	err := l.s.Init(conf.LBPort)
	if err !=nil {
		return err
	}
	l.backend = conf.Backend[0] + conf.BackendPort
	l.sCh = make(chan string)


	// Make referance to frontend servers \\
	for i := range(l.frontends) {
		f := &l.frontends[i]
		f.addr, err = net.ResolveUDPAddr("udp", conf.Frontends[i] + conf.FrontendPort)
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
			l.log.Fatal(err)
		}
		if string(msg) == "new_lease" {
			l.log.Print("New lease request from: ", remoteAddr)
			frontend, lease, err := l.NewClient(remoteAddr)
			if err != nil {
				l.log.Fatal(err)
			}
			go func() {
				l.s.Write([]byte(remoteAddr.String() + " " + lease.Format(time.UnixDate)), frontend.addr)
			}()
			l.s.Write([]byte(frontend.addr.String() + " " + lease.Format(time.UnixDate)), remoteAddr)
		} else if string(msg) == "frontend_up" {
			l.log.Print("NEW FRONTEND: ", remoteAddr.String())
			l.s.Write([]byte(l.backend), remoteAddr)
		}
	}
}

func (l *lb) NewClient(client *net.UDPAddr) (*lbFrontend, time.Time, error) {

	for i := range(l.frontends) {
		frontend := &l.frontends[i]
		if atomic.LoadInt32(frontend.up) == 1 && atomic.LoadInt32(frontend.numClients) < conf.MaxClientsPerFrontend {
			route := &route{
				client,
				frontend.addr,
				time.Now().Add(time.Duration(conf.LeaseTime) * time.Millisecond)}
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

			fmt.Printf("%s has %d clients\n", l.frontends[i].addr.String(), atomic.LoadInt32(l.frontends[i].numClients))
			// propegate information to frontends?
		}
	}
}


func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Handle command line arguments
	var confFile string
	flag.StringVar(&confFile, "c", "config.json", "Configuration file name") // src/config.json is default 
	flag.Parse()

	// Read configurations from file
    err := conf.GetConfig(confFile)
    if err != nil {
		log.Fatal(err)
	}

	lb := new(lb)
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	lb.log, err = logger.InitLogger("logs/lb/" + hostname)
	if err != nil {
		lb.log.Fatal(err)
	}

	err = lb.Init()
	if err != nil {
		lb.log.Fatal(err)
	}
	go lb.Info()
	lb.Serve()
}
