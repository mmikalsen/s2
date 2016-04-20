package main

import (
	"fmt"
	"errors"
	"log"
	"./lib/server"
	"./lib/config"
	"./lib/logger"
	"time"
	"os"
	"hash/crc32"
	"strconv"
	"net"
	"runtime"
	"strings"
	"github.com/streamrail/concurrent-map"
)

var(
	conf = new(config.Configuration)

	BLUE string = "\033[94m"
	GREEN string = "\033[92m"
	RED string = "\033[91m"
	ENDC string = "\033[0m"
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
	log *log.Logger
	hash Hash
	rCh chan bool
}

func (c *client) Init() (error) {
	var err error


	c.s = new(server.UDPServer)
	c.s.Init(conf.ClientPort)
	c.hash = crc32.ChecksumIEEE

	c.log.Print(conf.LB[0] + conf.LBPort)
	c.load_balancer, err = net.ResolveUDPAddr("udp", conf.LB[0] + conf.LBPort)
	if err != nil {
		return err
	}

	c.sCh = make(chan bool)
	c.rCh = make(chan bool)
	// index data structure
	c.index = cmap.New()

	// start listing to udp stream
	go c.recive()

	c.s.Write([]byte("new_lease"), c.load_balancer)
	c.log.Print("waiting for lb")

	// wait for conformation about frontend
	if ok := <-c.sCh; ok {
		c.log.Print("GOT " , c.frontend.String(), "as frontend")
		c.ttl = time.Duration(conf.ClientInitTTL) * time.Millisecond
		return nil
	} else {
		return errors.New("Could not contact the LoadBalancer")
	}
}

func (c *client) recive() {

	for {
		msg, remoteAddr, err := c.s.Read(64)
		if err != nil {
			c.log.Fatal()
		}
		//log.Print("MSG: ", string(msg))
		if remoteAddr.String() == c.load_balancer.String() {
			lease := strings.Split(string(msg), " ")
			c.frontend, err = net.ResolveUDPAddr("udp", lease[0])
			if err != nil {
				c.log.Fatal(err)
			}
			c.lease, err  = time.Parse(time.UnixDate, strings.Join(lease[1:], " "))
			if err != nil {
				c.log.Fatal(err)
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
					fmt.Printf(GREEN + "■" + ENDC)
					c.rCh <- true
					//log.Print(string(fetchedKey), ": ok - current ttl: ", c.ttl)
				} else {
					c.ttl += c.ttl/4
					fmt.Printf("■")
					//log.Print(string(fetchedKey), "ttl failed by:", t1.Sub(expire))
				}
			}
		}
	}
}

func (c *client) Request(count int) int{
	hostname, err := os.Hostname()
	if err != nil {
		c.log.Fatal(err)
	}
	ip, err := net.LookupIP(hostname)
	if err != nil {
		c.log.Fatal(err)
	}
	timeout := make(chan []byte)
	//go c.TimeoutMonitor(timeout)

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
		fmt.Printf(BLUE + "■" + ENDC)
		c.s.Write(key, c.frontend)

		select {
		case <- timeout:
			fmt.Printf(RED + "■" + ENDC)
			c.ttl = c.ttl + c.ttl/10
			//log.Print("TimeOut")
		case <- c.rCh:
			c.log.Print("recv")
		}

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
            fmt.Printf(RED + "■" + ENDC)
            c.ttl = c.ttl + c.ttl/10
            //log.Print("TimeOut")
        }
    }
}


func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

    err := conf.GetConfig("config.json")
    if err != nil {
		log.Fatal(err)
	}

	c := new(client)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	c.log, err = logger.InitLogger("logs/client/" + hostname)
	if err != nil {
		log.Fatal(err)
	}

	err = c.Init()
	if err != nil {
		c.log.Fatal(err)
	}

	c.log.Print("start requesting")
	count := 0
	for {
		count = c.Request(count)
		c.log.Print("Lease ran out!")
		c.s.Write([]byte("new_lease"), c.load_balancer)
		if ok := <- c.sCh; ok {
			c.log.Print("new frontend: ", c.frontend.String())
			continue
		}
	}


}
