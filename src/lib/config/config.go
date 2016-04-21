package config

import (
    "encoding/json"
    "os"
)

type Configuration struct {
	// HOSTS
    Hostfile string
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

func (config *Configuration) GetConfig(path string) error{

	file, err := os.Open(path)
    if err != nil {
      return err
    }
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
      return err
	}

    return err
}
