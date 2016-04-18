package config

import (
    "encoding/json"
    "os"
    "fmt"
)

type Configuration struct {
	// HOSTS
    LB    []string
    Frontends []string
    Backend []string
    // PORTS
    ClientPort string
    LBPort string
    FrontendPort string
    BackendPort string
    // Vars
    MaxClientsPerFrontend int32
    LeaseTime int32
    ClientInitTTL int32
    FrontendInitTTL int32
}

func (config *Configuration) GetConfig(path string){

	file, _ := os.Open(path)
	decoder := json.NewDecoder(file)
	err := decoder.Decode(config)
	if err != nil {
	  fmt.Println("Config error:", err)
	}
}