package topology

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ServerInfo ...
type ServerInfo struct {
	NodeName   string `json:"node_name"`
	PortNumber int    `json:"port_number"`
	IpAddress  string `json:"ip_address"`
}

// NewServerInfo ...
func NewServerInfo(name string, port int, ipAddress string) ServerInfo {
	return ServerInfo{
		NodeName:   name,
		PortNumber: port,
		IpAddress:  ipAddress,
	}
}

// String ...
func (s ServerInfo) String() string {
	return fmt.Sprintf("[%s] %s:%d", s.NodeName, s.IpAddress, s.PortNumber)
}

// ReadAll ...
func ReadAll(filePath string) ([]ServerInfo, error) {
	fPtr, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(fPtr)
	if err != nil {
		return nil, err
	}

	var servers []ServerInfo
	err = json.Unmarshal(data, &servers)
	if err != nil {
		return nil, err
	}
	return servers, nil
}
