package main

import (
	"fmt"
  "bufio"
  "os"
	"./lib/config"
	"net"
	"flag"
  "strconv"
)

var(
	conf = new(config.Configuration)
  clients []string
  numClients int
  activeClients int
)


func sendCommand(command string, addr string){

  fmt.Printf("send %s to %s\n", command, addr)
  conn, err := net.Dial("udp", addr)

  if err != nil {
    fmt.Printf("Send command error: %v\n", err)
    return
  }
  fmt.Fprintf(conn, command)
  conn.Close()
}

func addClients(hostfile string){
  var i int
  inFile, _ := os.Open(hostfile)
  defer inFile.Close()
  scanner := bufio.NewScanner(inFile)
  scanner.Split(bufio.ScanLines)

  for scanner.Scan() {
    clients = append(clients, scanner.Text())
    i++
    if(i > 25){break}
  }
  numClients = i
}

func startClients(num int){
  var i int

  for i = range clients{
    if i >= numClients {break}

    if num == -1 {
      sendCommand("kill", clients[i]+conf.ClientPort)
    }else if num == 0 {
      sendCommand("stop", clients[i]+conf.ClientPort)
    }else if i <= num-1 {
      sendCommand("start", clients[i]+conf.ClientPort)
    }else{
      sendCommand("stop", clients[i]+conf.ClientPort)
    }

  }

}



func main() {
  var err error
  var str string


  clients = make([]string, 0)
  fmt.Println(numClients)


  // Handle command line arguments
  var confFile string
  flag.StringVar(&confFile, "c", "config.json", "Configuration file name") // src/config.json is default
  flag.Parse()

  // Read configurations from file
  err = conf.GetConfig(confFile)
  if err != nil {
    fmt.Println(err)
    return
  }

  addClients(conf.Hostfile)


  for {
    fmt.Printf("\nSet number of clients (%d)\n", activeClients)
    reader := bufio.NewReader(os.Stdin)
    input, _ := reader.ReadString('\n')

    str = string([]byte(input))
    activeClients, err = strconv.Atoi(str[:len(str)-1])
    if err != nil {
      fmt.Println(err)
    }
    if (activeClients > 24) {activeClients = 24}
    if (activeClients > len(clients)) {activeClients = len(clients)}
    if (activeClients < -1) {activeClients = 0}

    startClients(activeClients)

  }
}
