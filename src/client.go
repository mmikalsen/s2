package main

import (
   "fmt"
   "errors"
    "log"
    "./lib/server"
    "./lib/config"
    "time"
    "os"
	"hash/crc32"
    "strconv"
    "net"
    "runtime"
	"strings"
	"github.com/streamrail/concurrent-map"
)


type Hash func(data []byte) uint32
type client struct {
    s *server.UDPServer
    load_balancer *net.UDPAddr
    frontend *net.UDPAddr
	lease time.Time
    sCh chan bool
    ttl time.Duration
    index  cmap.ConcurrentMap
	hash Hash
}

func (c *client) Init(port string, load_balancer_addr string) (error) {
    var err error


   	conf := new(config.Configuration)
   	conf.GetConfig("config.json")
   	fmt.Println(conf.Frontends)


    c.s = new(server.UDPServer)
    c.s.Init(port)
	c.hash = crc32.ChecksumIEEE

    c.load_balancer, err = net.ResolveUDPAddr("udp", load_balancer_addr)
    if err != nil {
        return err
    }

    c.sCh = make(chan bool)
    // index data structure
    c.index = cmap.New()

    // start listing to udp stream
    go c.recive()

	c.s.Write([]byte("new_lease"), c.load_balancer)
	log.Print("waiting for lb")

	// wait for conformation about frontend
	if ok := <- c.sCh; ok {
		log.Print("GOT " , c.frontend.String(), "as frontend")
		c.ttl = 10 * time.Millisecond
		return nil
	} else {
		return errors.New("Could not contact the LoadBalancer")
	}
}

func (c *client) recive() {

    for {
        msg, remoteAddr, err := c.s.Read(64)
        if err != nil {
            log.Fatal()
        }
		//log.Print("MSG: ", string(msg))
		if remoteAddr.String() == c.load_balancer.String() {
			lease := strings.Split(string(msg), " ")
			c.frontend, err = net.ResolveUDPAddr("udp", lease[0])
			if err != nil {
				log.Fatal(err)
			}
			c.lease, err  = time.Parse(time.UnixDate, strings.Join(lease[1:], " "))
			if err != nil {
				log.Fatal(err)
			}
			c.sCh <- true
		} else if remoteAddr.String() == c.frontend.String() {
			t1 := time.Now()
			if val, ok := c.index.Get(string(msg)); ok {

				/* Start a goroutine to remove the key from trie */
				go func() {
					c.index.Remove(string(msg))
				}()

				expire, _ := val.(time.Time)
				if expire.After(t1) {
					c.ttl -= c.ttl/16
					fmt.Printf("\x1b[32;1m■")
					//log.Print(string(fetchedKey), ": ok - current ttl: ", c.ttl)
				} else {
					c.ttl += c.ttl/4
					fmt.Printf("\x1b[0m■")
					//log.Print(string(fetchedKey), "ttl failed by:", t1.Sub(expire))
				}
			}
        }
    }
}

func (c *client) Request(count int) int{
    hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	ip, err := net.LookupIP(hostname)
	if err != nil {
		log.Fatal(err)
	}
    timeout := make(chan []byte, 100)
    go c.TimeoutMonitor(timeout)

    for i := count; ;i++{
        key := []byte(strconv.FormatUint(uint64(c.hash(ip[0])), 10) + " " + strconv.Itoa(i))
        ttl := time.Now().Add(c.ttl)
        go func() {
            tKey := key
            time.Sleep(c.ttl)
            timeout <- tKey
        }()

        c.index.Set(string(key),ttl)
        //log.Print("Sent: ", key, "- expire: ", ttl)
        fmt.Printf("\x1b[34;1m■")
        c.s.Write(key, c.frontend)
		time.Sleep(2 * time.Second)
		t1 := time.Now()
		if c.lease.After(t1) {
			continue
		} else {
			return i + 1
		}
    }
}

func(c *client) TimeoutMonitor(ch chan []byte) {
    for {
        signal := <-ch
        if _, ok := c.index.Get(string(signal)); ok {
            fmt.Printf("\x1b[31;1m■")
            c.ttl = c.ttl + c.ttl/10
            //log.Print("TimeOut")
        }
    }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    c := new(client)
	err := c.Init(":9000", "compute-8-1:9000")
	if err != nil {
		log.Fatal(err)
	}

    log.Print("start requesting")
	count := 0
	for {
		count = c.Request(count)
		//log.Print("Lease ran out! old: ", c.frontend)
		c.s.Write([]byte("new_lease"), c.load_balancer)
		if ok := <- c.sCh; ok {
			//log.Print("Switching too ", c.frontend)
			continue
		}
	}

}
