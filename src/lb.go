package main

import  (
  "log"
  "net"
  "./lib/server"
  "strings"
  "strconv"
  "sync/atomic"
  "unsafe"
  "runtime"
  "time"
)

var (
	LEASETIME = time.Second * 10
)


type lbClient struct {
	addr *net.UDPAddr
	hash uint64
	lease time.Time
}

type lbFrontend struct {
	addr *net.UDPAddr
	clients []unsafe.Pointer
	numClients *int32
	up bool
}


type lb struct {
	fs []lbFrontend
	s *server.UDPServer
	hash Hash
}

func (l *lb) Init(port string) error {
	l.s = new(server.UDPServer)
	err := l.s.Init(port)
	if err != nil {
		return err
	}

	l.fs = make([]lbFrontend, 3)
	for i := range(l.fs) {
		f := &l.fs[i]
		f.addr, err = net.ResolveUDPAddr("udp", "compute-10-" + strconv.Itoa((i + 1)) +":9001")

		f.up = true
		f.clients = make([]unsafe.Pointer, 7)
		f.numClients = new(int32)

		log.Print("adding: compute-10-" + strconv.Itoa((i + 1)) + ":9001 - ", f.addr.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *lb) Serve() {
	for {
		fetchedKey, remoteAddr, err := l.s.Read(32)
		if err != nil {
			log.Fatal(err)
		}


		go func() {
			tmpKey := fetchedKey
			tmpAddr := remoteAddr
			hash, err := strconv.ParseUint(strings.Split(string(fetchedKey), " ")[0], 10, 64)
			if err != nil {
				log.Fatal(err)
			}

			var c *lbClient

			// From Frontend \\
			for _, f := range(l.fs) {
				if f.addr.String() == tmpAddr.String() {
					for i, _ := range(f.clients) {
						c = (*lbClient)(atomic.LoadPointer(&f.clients[i]))
						if c != nil {
							if c.hash == hash {
								n, err := l.s.Write(tmpKey, c.addr)
								if n != len(tmpKey) || err != nil {
									log.Fatal(err)
								}
								return
							}
						}
					}
				}
			}

			// From Client \\
			f := &l.fs[hash % uint64(len(l.fs))]
			log.Print("Frontend: ", f.addr.String())
			if f.up {
				// Check if client is in the frontend 'bucket' \\
				for i, _ := range(f.clients) {
					c = (*lbClient)(atomic.LoadPointer(&f.clients[i]))
					if c != nil {
						if c.hash == hash {
							n, err := l.s.Write(tmpKey, f.addr)
							if n != len(tmpKey) || err != nil {
								log.Fatal(err)
							}
							return
							// check leasetime
							if time.Now().After(c.lease) {
								log.Print("lease expired!")
								// TODO
							}
						}
					// Reached the end of bucket, insert the new client \\
					} else if i < 7 && int32(i) == atomic.LoadInt32(f.numClients) {
						log.Print("New Client: ", tmpAddr, " - frontend - ", f.addr)
						c := &lbClient{tmpAddr, hash, time.Now().Add(LEASETIME)}
						cUN := unsafe.Pointer(c)
						atomic.StorePointer(&f.clients[atomic.LoadInt32(f.numClients) + 1], cUN)
						atomic.AddInt32(f.numClients, 1)

						n, err := l.s.Write(tmpKey, f.addr)
						if n != len(tmpKey) || err != nil {
							log.Fatal(err)
						}
						return

					} else if i >= 7 {
						// TODO something smart here too
					}
				}
			} else {
				// TODO do something smart //

			}
			return
		}()
	}
}



func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	l := new(lb)
	err := l.Init(":9000")
	if err != nil {
		log.Fatal(err)
	}
	l.Serve()
}
