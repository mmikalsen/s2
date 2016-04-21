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
    "flag"
    ui "github.com/gizak/termui"
    _"math/rand"
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
    eCh chan string
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
    l.eCh = make(chan string, 10)


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
            frontend, lease, err := l.NewClient(remoteAddr)
            if err != nil {
                l.log.Fatal(err)
            }
            go func() {
                l.s.Write([]byte(remoteAddr.String() + " " + lease.Format(time.UnixDate)), frontend.addr)
            }()
            l.s.Write([]byte(frontend.addr.String() + " " + lease.Format(time.UnixDate)), remoteAddr)

            l.eCh <- remoteAddr.String() + " Assigned to " + frontend.addr.String()
            l.log.Print(remoteAddr.String() + " Assigned to " + frontend.addr.String())

        // FIXME -- Make the Frontend contact the LB when starting up -- \\
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
                    l.eCh <- client.String() + " - Lease ran out!"
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


    func (l *lb) Ui() {
        if err := ui.Init(); err != nil {
            l.log.Fatal(err)
        }
        defer ui.Close()

        bc := ui.NewBarChart()
        bclabels := []string{"f0", "f1", "f2"}
        data := make([]int, 3)


        bc.BorderLabel = "Frontend Load"
        bc.Data = data
        bc.Width = 25
        bc.Height = 20
        bc.DataLabels = bclabels
        bc.TextColor = ui.ColorGreen
        bc.BarColor = ui.ColorRed
        bc.NumColor = ui.ColorYellow

        events := ui.NewList()
        events.BorderLabel = "Events"
        events.Items = make([]string, 18)
        events.Height = 20
        eventCount := 0

        // TODO sparkline - History of load

        // build layout
        ui.Body.AddRows(
            ui.NewRow(
                    ui.NewCol(2, 0, bc),
                    ui.NewCol(10, 0, events),
            ),
        )
        // calculate layout
        ui.Body.Align()


        ui.Handle("/sys/kbd/q", func(ui.Event) {
            ui.StopLoop()
            // TODO might be smart to inform loadbalancer
        })

        ui.Handle("/timer/1s", func(e ui.Event) {
            for i, front := range(l.frontends) {
                bc.Data[i] = int(atomic.LoadInt32(front.numClients))
            }

            for {
                select {
                case m := <- l.eCh:
                    if eventCount >= 18 {
                         events.Items = make([]string, 18)
                         eventCount = 0
                    }
                    events.Items[eventCount] = m
                    eventCount += 1
                default:
                    ui.Render(ui.Body)
                    return
                }
                ui.Render(ui.Body)
            }
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
        go lb.Serve()
        lb.Ui()
    }
