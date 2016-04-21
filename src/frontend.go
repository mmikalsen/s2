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
        f.log.Fatal(err)
    }
    f.numClients = new(int32)
    atomic.StoreInt32(f.numClients, 0)

    f.sCh = make(chan string)
    go f.recive()

    f.log.Println(f.load_balancer.String())

    // send init message to lb
    //f.s.Write([]byte("frontend_up"), f.load_balancer) //TODO: invalid memory address

    //msg := <-f.sCh

    f.backend_addr = conf.Backend[0] + conf.BackendPort

    //f.s.Write([]byte("ACK"), f.load_balancer)

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
                time.Sleep(lease.Sub(time.Now()))
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

    go func() {
        time.Sleep(ttl)
        timeoutCh <- true
    }()
    go func() {
        resp, err := http.Get("http://" + f.backend_addr)
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
            return
        }
        f.s.Write(key, addr)
        f.ttl -= f.ttl/20
    case <- timeoutCh:
        // timeout
        f.ttl += f.ttl/10
        f.log.Print("timeout")
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

    ttl := ui.NewList()
    ttl.BorderLabel = "TTL"
    ttl.Items = []string{f.ttl.String()}
    ttl.Height = 10


    ttlh := ui.NewLineChart()
    ttlh.BorderLabel = "TTL"
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
            ui.NewCol(6, 0, ttl),
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
        i := 0
        clients.Items = make([]string, conf.MaxClientsPerFrontend)
        clients.BorderLabel = "Clients - " + strconv.Itoa(int(atomic.LoadInt32(f.numClients)))

        limit := int(atomic.LoadInt32(f.numClients))
        for item := range(f.clients.Iter()) {

            clients.Items[i] = item.Key + " : " + item.Val.(time.Time).Format(time.Stamp)
            i++
            if i >= limit {
                break
            }
        }

        ttl.Items = []string{f.ttl.String()}
        ttlh.Data = ttlh.Data[1:]
        ttlh.Data = append(ttlh.Data, f.ttl.Seconds())
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
