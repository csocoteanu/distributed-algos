package vectorclocks

import (
	"distributed-algos/topology"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBroadcast(t *testing.T) {
	// create server topology
	serverInfos := []topology.ServerInfo{
		topology.NewServerInfo("A", 8080, ""),
		topology.NewServerInfo("B", 8081, ""),
		topology.NewServerInfo("C", 8082, ""),
	}
	var peers []*VectorClocksPeer
	for i, sInfo := range serverInfos {
		var peerTopology []topology.ServerInfo
		for j := 0; j < i; j++ {
			peerTopology = append(peerTopology, serverInfos[j])
		}
		if i < len(serverInfos) {
			peerTopology = append(peerTopology, serverInfos[i+1:]...)
		}

		peer := NewVectorClocksPeer(sInfo, peerTopology)
		peers = append(peers, peer)
	}

	// start the servers
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

	// each server broadcasts it's own message in the network
	errChan := make(chan broadcastResult, len(peers))
	wg := sync.WaitGroup{}
	wg.Add(len(peers))
	for i, peer := range peers {
		go func(index int, p *VectorClocksPeer) {
			defer wg.Done()

			message := fmt.Sprintf("message-%s", p.me.NodeName)
			sleepTime := time.Duration((len(peers) - index) * 300 * int(time.Millisecond))

			time.Sleep(sleepTime)
			err := p.Broadcast(message)
			errChan <- broadcastResult{
				err:      err,
				nodeName: p.WhoAmI(),
			}
		}(i, peer)
	}

	// wait for broadcast to finish on all nodes
	wg.Wait()
	for i := 0; i < len(peers); i++ {
		status := <-errChan
		if status.err != nil {
			t.Fatalf("Unexpected broadcast on node=%s error: %v", status.nodeName, status.err)
		}
	}

	// check all nodes receive the messages in the exact same order
	expectMessageOrder := []string{
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
