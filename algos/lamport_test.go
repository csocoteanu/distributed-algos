package algos

import (
	"distributed-algos/topology"
	"fmt"
	"testing"
	"time"
)

func TestBroadcast(t *testing.T) {
	serverInfos := []topology.ServerInfo{
		topology.NewServerInfo("A", 8080, ""),
		topology.NewServerInfo("B", 8081, ""),
		topology.NewServerInfo("C", 8082, ""),
		topology.NewServerInfo("D", 8083, ""),
	}

	var peers []*LamportPeer
	for i, sInfo := range serverInfos {
		var peerTopology []topology.ServerInfo
		for j := 0; j < i; j++ {
			peerTopology = append(peerTopology, serverInfos[j])
		}
		if i < len(serverInfos) {
			peerTopology = append(peerTopology, serverInfos[i+1:]...)
		}

		peer := NewLamportPeer(sInfo, peerTopology)
		peers = append(peers, peer)
	}

	for _, p := range peers {
		go p.Start()
	}
	defer func() {
		for _, p := range peers {
			p.Stop()
		}
	}()
	// TODO: replace time.Sleep
	time.Sleep(5 * time.Second)

	for i, peer := range peers {
		go func(index int, p *LamportPeer) {
			message := fmt.Sprintf("message-%s", p.me.NodeName)
			sleepTime := time.Duration((len(peers) - index) * 300 * int(time.Millisecond))

			time.Sleep(sleepTime)
			p.Broadcast(message)

		}(i, peer)
	}
	// TODO: replace time.Sleep
	time.Sleep(5 * time.Second)

	expectMessageOrder := []string{
		"message-D",
		"message-C",
		"message-B",
		"message-A",
	}
	for _, p := range peers {
		messages := p.Messages()
		for i, m := range messages {
			if expectMessageOrder[i] != m.Body {
				t.Fatalf("Unexpected message order for node=%s. total order is=%+v, but got=%+v",
					p.WhoAmI(),
					expectMessageOrder,
					messages)
			}
		}
	}
}
