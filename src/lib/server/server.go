package server


import (
    "net"
    "net/http"
    "errors"
	"time"
)

type UDPServer struct {
    Conn *net.UDPConn
    LocalAddr *net.UDPAddr
	Client http.Client
}

func (s *UDPServer) Init(port string) error {

    serverAddr, err := net.ResolveUDPAddr("udp", port)
    s.LocalAddr = serverAddr
    if err != nil {
        return err
    }

    s.Conn, err = net.ListenUDP("udp", serverAddr)
    if err != nil {
        return err
    }

	s.Client = http.Client{
		    Timeout: time.Duration(0),
	}

    return nil
}

func (s *UDPServer) Write(msg []byte, remoteAddr *net.UDPAddr) (int, error) {
    n, err := s.Conn.WriteToUDP(msg, remoteAddr)
    return n, err
}

func (s *UDPServer) Read(b int) ([]byte, *net.UDPAddr, error) {
    buf := make([]byte, b)
    n, addr, err := s.Conn.ReadFromUDP(buf)
    if err != nil {
         return nil, nil, err
    }
    return buf[0:n], addr, nil
}

func(s *UDPServer) http_request(addr string) (error) {
    resp, err := s.Client.Get(addr)
    if err != nil {
         return err
    }

    if status := resp.Header.Get("status"); status != "200" {
        return errors.New(status + "not valied HTTP status")
    }
    return nil
}
