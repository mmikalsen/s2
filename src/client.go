package main

import (
	"fmt"
	//"errors"
	"log"
	"./lib/server"
	"./lib/config"
	"./lib/logger"
	"time"
	"os"
	"strconv"
	"net"
	"runtime"
	"strings"
	"github.com/streamrail/concurrent-map"
	"flag"
	_"github.com/beevik/ntp"
	//ui "github.com/gizak/termui"
)

var(
	conf = new(config.Configuration)
	ntpServer = "ntp.uit.no"
	BLUE string = "\033[94m"
	GREEN string = "\033[92m"
	RED string = "\033[91m"
	YELLOW string = "\033[33m"
	ENDC string = "\033[0m"
	KILL string = "kill"
	STOP string = "stop"
	START string = "start"
)

type Hash func(data []byte) uint32
type client struct {
	s *server.UDPServer
	load_balancer *net.UDPAddr
	frontend *net.UDPAddr
	lease time.Time
	newLeaseCh chan bool
	ttl time.Duration
	index  cmap.ConcurrentMap
	log *log.Logger
	msgRecivedCh chan bool
	running bool
	startupSignalCh chan bool
	localip []net.IP


	nRequests int
	nTimeouts int

}

func (c *client) Init() (error) {
	var err error


	c.s = new(server.UDPServer)
	c.s.Init(conf.ClientPort)

	c.nRequests = 0
	c.nTimeouts = 0

	c.log.Print(conf.LB[0] + conf.LBPort)
	c.load_balancer, err = net.ResolveUDPAddr("udp", conf.LB[0] + conf.LBPort)
	if err != nil {
		return err
	}

	hostname, err := os.Hostname()
	if err != nil {
		c.log.Fatal(err)
	}
	c.localip, err = net.LookupIP(hostname)
	if err != nil {
		c.log.Fatal(err)
	}

	c.running = true
	c.newLeaseCh = make(chan bool)
	c.msgRecivedCh = make(chan bool)
	c.startupSignalCh = make(chan bool)
	// index data structure
	c.index = cmap.New()
	c.ttl = time.Duration(conf.ClientInitTTL) * time.Millisecond

	// start listing to udp stream
	go c.recive()

	return nil
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
			c.log.Print("GOT " , c.frontend.String(), "as frontend")
			c.newLeaseCh <- true
		} else if remoteAddr.String() == c.frontend.String() {
			t1 := time.Now()
			if val, ok := c.index.Get(string(msg)); ok {

				/* Start a goroutine to remove the key from trie */
				go func() {
					c.index.Remove(string(msg))
				}()

				expire, _ := val.(time.Time)
				if expire.After(t1) {
					c.ttl -= time.Duration(20 * time.Millisecond)
					fmt.Printf(GREEN + "■" + ENDC)
					c.msgRecivedCh <- true
					//log.Print(string(fetchedKey), ": ok - current ttl: ", c.ttl)
				} else {
					c.ttl += time.Duration(100 * time.Millisecond)
					fmt.Printf("■")
					//log.Print(string(fetchedKey), "ttl failed by:", t1.Sub(expire))
				}
			}
		} else if string(msg) == START {
			c.log.Print("START signal recived, now running")
			c.running = true
			c.startupSignalCh <- true
		} else if string(msg) == STOP {
			c.log.Print("STOP signal recived, now stopped - client ttl is reset")
			c.ttl = time.Duration(conf.ClientInitTTL) * time.Millisecond
			c.running = false
		} else if string(msg) == KILL {
			c.log.Print("KILL signal recived, shutting down")
			os.Exit(0)
		} else {
			c.log.Print("Unknown message recived", string(msg))
		}
	}
}

func (c *client) Request(count int) int{
	timeout := make(chan []byte)
	c.nRequests++

	//for i := count; ;i++{
	key := []byte(c.localip[0].String() + " " + strconv.Itoa(count))
	t1 := time.Now()
	ttl := t1.Add(c.ttl)

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
		c.ttl += time.Duration(300 * time.Millisecond)
		c.nTimeouts++
		c.log.Print("TimeOut")
	case <- c.msgRecivedCh:
		c.log.Print("recv")
	}
	count ++

	return count

}

func (c *client) CheckLease() bool{


	/* t1, err := ntp.Time(ntpServer)
	if err != nil {
		log.Fatal(err)
	}
	*/
	t1 := time.Now()
	c.log.Print(c.lease.String() + " " + t1.String())
	if c.lease.After(t1) {
		return true
	} else {
		return false
	}
}

func (c *client) RenewLease(retry bool,timeoutMS int) bool{
	timeout := make(chan bool)

	for retry && c.running{
		c.log.Print("requesting lease from: " + c.load_balancer.String())
		c.s.Write([]byte("new_lease"), c.load_balancer)
		c.log.Print("new lease request")
		go func() {
			time.Sleep(time.Duration(200 * time.Millisecond))
			timeout <- true
		}()

		select{
		case <-c.newLeaseCh:
			c.log.Print("new frontend lease recived: ", c.frontend.String())
			return true
		case <-timeout:
			c.log.Print("lease request timeout ")
			continue
		}
	}
	return false
}

func (c *client) Metric() {
	var tot_ttl time.Duration
	for i := 1; ;i++ {
		time.Sleep(10 * time.Second)

		tot_ttl += c.ttl
		c.log.Print("Average ttl: " + (tot_ttl / time.Duration(i)).String())
		c.log.Print("Timeout % " + strconv.FormatFloat(float64(c.nTimeouts)/float64(c.nRequests) * 100, 'f', -1, 64))
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

	c := new(client)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	go c.Metric()

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
		if !c.running {
			if <-c.startupSignalCh {} //wait for start signal
		}

		if c.CheckLease() {
			count = c.Request(count)
		} else {
			c.RenewLease(true,10)
		}

	}
}
