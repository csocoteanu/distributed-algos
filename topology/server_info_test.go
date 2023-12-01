package topology

import (
	"os"
	"testing"
)

func TestNewServerInfo(t *testing.T) {
	name := "testNode"
	port := 8080
	ip := "127.0.0.1"

	server := NewServerInfo(name, port, ip)

	if server.NodeName != name {
		t.Errorf("Expected NodeName %s, got %s", name, server.NodeName)
	}

	if server.PortNumber != port {
		t.Errorf("Expected PortNumber %d, got %d", port, server.PortNumber)
	}

	if server.IpAddress != ip {
		t.Errorf("Expected IpAddress %s, got %s", ip, server.IpAddress)
	}
}

func TestReadAll(t *testing.T) {
	filePath := "./test.json"
	data := `[{"node_name":"testNode","port_number":8080,"ip_address":"127.0.0.1"}]`
	err := os.WriteFile(filePath, []byte(data), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(filePath)

	servers, err := ReadAll(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 1 {
		t.Fatalf("Expected 1 server, got %d", len(servers))
	}

	server := servers[0]
	if server.NodeName != "testNode" {
		t.Errorf("Expected NodeName %s, got %s", "testNode", server.NodeName)
	}

	if server.PortNumber != 8080 {
		t.Errorf("Expected PortNumber %d, got %d", 8080, server.PortNumber)
	}

	if server.IpAddress != "127.0.0.1" {
		t.Errorf("Expected IpAddress %s, got %s", "127.0.0.1", server.IpAddress)
	}
}

func TestReadAll_TopologyRepoFile(t *testing.T) {
	servers, err := ReadAll("./topology.json")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedServers := map[string]ServerInfo{
		"server-A": {
			NodeName:   "server-A",
			PortNumber: 9000,
			IpAddress:  "127.0.0.1",
		},
		"server-B": {
			NodeName:   "server-B",
			PortNumber: 9001,
			IpAddress:  "127.0.0.1",
		},
		"server-C": {
			NodeName:   "server-C",
			PortNumber: 9002,
			IpAddress:  "127.0.0.1",
		},
		"server-D": {
			NodeName:   "server-D",
			PortNumber: 9003,
			IpAddress:  "127.0.0.1",
		},
	}

	if len(servers) != 4 {
		t.Errorf("Expected 4 servers, got %d", len(servers))
	}

	for _, server := range servers {
		expectedServer, ok := expectedServers[server.NodeName]
		if !ok {
			t.Errorf("Unexpected server %s", server.NodeName)
		}
		if server.PortNumber != expectedServer.PortNumber {
			t.Errorf("Expected PortNumber %d, got %d", expectedServer.PortNumber, server.PortNumber)
		}
		if server.IpAddress != expectedServer.IpAddress {
			t.Errorf("Expected IpAddress %s, got %s", expectedServer.IpAddress, server.IpAddress)
		}
		if server.NodeName != expectedServer.NodeName {
			t.Errorf("Expected NodeName %s, got %s", expectedServer.NodeName, server.NodeName)
		}
	}
}
