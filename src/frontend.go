package main

import (
	"net"
	"log"
	"time"
	"strings"
	"runtime"
	"os"
	"./lib/server"
	"./lib/config"
	"./lib/logger"
	"net/http"
	"github.com/streamrail/concurrent-map"
	"sync/atomic"
	"flag"
	ui "github.com/gizak/termui"
	_"github.com/beevik/ntp"
	"strconv"
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
	numClients *int32
	sCh chan string
	ttl time.Duration
	log *log.Logger
	nRequests int
	nTimeouts int
}

var (
	conf = new(config.Configuration)
	ntpServer = "ntp.uit.no"
)

func (f *Frontend) Init() {
	var err error
	f.s = new(server.UDPServer)
	f.s.Init(conf.FrontendPort)
	f.ttl = time.Duration(conf.FrontendInitTTL) * time.Millisecond

	f.load_balancer, err = net.ResolveUDPAddr("udp", conf.LB[0] + conf.LBPort)
	if err != nil {
		f.log.Fatal(err)
	}
	f.numClients = new(int32)
	atomic.StoreInt32(f.numClients, 0)

	f.sCh = make(chan string)
	go f.recive()

	f.log.Println(f.load_balancer.String())
	f.backend_addr = conf.Backend[0] + conf.BackendPort
	f.clients = cmap.New()
}



func (f *Frontend) recive() {

	for {
		body, remoteAddr, err := f.s.Read(64) //TODO: Some runtime error here
		if err != nil {
			f.log.Fatal(err)
		}
		if remoteAddr.String() == f.load_balancer.String() {
			// msg: "clientAddr lease"
			//log.Print("GOT MSG: load_balancer - " + string(body))
			if (atomic.LoadInt32(f.numClients) >= int32(conf.MaxClientsPerFrontend)) {
				f.log.Print("Client limit reached. Aborting receive")
				return
			}
			msg := strings.Split(string(body), " ")

			// The lease is multiple "words" in the string, so join them together again
			lease, err  := time.Parse(time.UnixDate, strings.Join(msg[1:], " "))
			if err != nil {
				f.log.Fatal(err)
			}
			clientAddr := msg[0]
			f.clients.Set(clientAddr, lease)
			atomic.AddInt32(f.numClients, 1)

			go func() {
				// Remove the client once the lease runs out
				/* t1, err := ntp.Time(ntpServer)
				if err != nil {
					f.log.Fatal(err)
				}
				*/
				t1 := time.Now()
				time.Sleep(lease.Sub(t1))
				f.clients.Remove(clientAddr)
				atomic.AddInt32(f.numClients, -1)
			}()
		} else {
			_, ok := f.clients.Get(remoteAddr.String())
			if !ok {
				f.log.Print("Lease ran out/not authorized client")
				return
			}

			//log.Print("Sending request: ", string(body), " to backend")
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
	f.nRequests++

	go func() {
		time.Sleep(ttl)
		timeoutCh <- true
	}()
	go func() {
		f.s.Client.Timeout = f.ttl
		resp, err := f.s.Client.Get("http://" + f.backend_addr)
		if err != nil {
			f.log.Print(err)
			return
		}
		defer resp.Body.Close()
		responseCh <- *resp
	}()

	select {
	case r := <-responseCh:
		// got response before timeout
		if r.StatusCode != 200 {
			f.nTimeouts++
			return
		}
		f.s.Write(key, addr)
		f.ttl -= 20 * time.Millisecond
	case <- timeoutCh:
		// timeout
		f.ttl += 100 * time.Millisecond
		f.log.Print("timeout")
		f.nTimeouts++
	}

}

func (f *Frontend) runtime() {

	if err := ui.Init(); err != nil {
		f.log.Fatal(err)
	}
	defer ui.Close()

	clients := ui.NewList()
	clients.BorderLabel = "Clients - " + strconv.Itoa(int(atomic.LoadInt32(f.numClients)))
	clients.Items = make([]string, conf.MaxClientsPerFrontend)
	clients.Height = 10

	timeouts := ui.NewLineChart()
	timeouts.BorderLabel = "% of timeouts"
	timeouts.Data = make([]float64, 110)
	timeouts.Width = 50
	timeouts.Height = 15
	timeouts.X = 0
	timeouts.Y = 0
	timeouts.AxesColor = ui.ColorWhite
	timeouts.LineColor = ui.ColorRed | ui.AttrBold

	var ttlh_tot time.Duration
	ttlh := ui.NewLineChart()
	ttlh.BorderLabel = "TTL: current:" + f.ttl.String() + " average: " + ttlh_tot.String()
	ttlh.Data = make([]float64, 220)
	ttlh.Width = 50
	ttlh.Height = 17
	ttlh.X = 0
	ttlh.Y = 0
	ttlh.AxesColor = ui.ColorWhite
	ttlh.LineColor = ui.ColorGreen | ui.AttrBold


	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(6, 0, clients),
			ui.NewCol(6, 0, timeouts),
		),
		ui.NewRow(
			ui.NewCol(12, 0, ttlh),
		),
	)

	ui.Body.Align()

	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})

	ui.Handle("/timer/1s", func(e ui.Event) {
		t := e.Data.(ui.EvtTimer)
		i := 0
		clients.Items = make([]string, conf.MaxClientsPerFrontend)
		clients.BorderLabel = "Clients - " + strconv.Itoa(int(atomic.LoadInt32(f.numClients)))

		limit := int(atomic.LoadInt32(f.numClients))
		for item := range(f.clients.Iter()) {

			clientHost, err := net.LookupAddr(strings.Split(item.Key, ":")[0])
			if err != nil {
				f.log.Print(err)
			}
			clients.Items[i] = clientHost[0] + " : " + item.Val.(time.Time).Format(time.Stamp)
			i++
			if i >= limit {
				break
			}
		}

		timeouts.Data = timeouts.Data[1:]
		tmp_timeouts := (float64(f.nTimeouts) / float64(f.nRequests)) * 100
		timeouts.Data = append(timeouts.Data, tmp_timeouts)
		timeouts.BorderLabel = strconv.FormatFloat(tmp_timeouts, 'f', -1, 64) + " % timeouts"

		ttlh_tot += f.ttl
		ttlh.Data = ttlh.Data[1:]
		ttlh.Data = append(ttlh.Data, f.ttl.Seconds())
		ttlh.BorderLabel = "TTL: current:" + f.ttl.String() + " average:" + (ttlh_tot/time.Duration(t.Count)).String()
		ui.Render(ui.Body)
	})

	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		ui.Body.Width = ui.TermWidth()
		ui.Body.Align()
		ui.Render(ui.Body)
	})

	ui.Loop()
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


	frontend := new(Frontend)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	frontend.log, err = logger.InitLogger("logs/frontend/" + hostname)
	if err != nil {
		log.Fatal(err)
	}
	frontend.Init()
	frontend.runtime()
}
