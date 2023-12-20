package vectorclocks

import (
	"context"
	"distributed-algos/tcp"
	"distributed-algos/topology"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

const (
	broadCastPeerResponse = "V-ACK"
)

// VectorClocksMessage ...
type VectorClocksMessage struct {
	Origin     topology.ServerInfo `json:"origin"`
	Body       string              `json:"body"`
	Timestamps map[string]int      `json:"timestamp"`
}

// Serialize ...
func Serialize(m VectorClocksMessage) ([]byte, error) {
	return json.Marshal(m)
}

// Deserialize ...
func Deserialize(b []byte, m *VectorClocksMessage) error {
	if m == nil {
		return fmt.Errorf("unexpected nil pointer received")
	}

	return json.Unmarshal(b, m)
}

type vectorClocksClient struct {
	nodeInfo topology.ServerInfo
	peer     *VectorClocksPeer
	client   *tcp.Client
}

func newVectorClocksClient(
	nodeInfo topology.ServerInfo,
	peer *VectorClocksPeer,
	client *tcp.Client) *vectorClocksClient {

	return &vectorClocksClient{
		nodeInfo: nodeInfo,
		peer:     peer,
		client:   client,
	}
}

func (c *vectorClocksClient) sendBroadcast(strMessage string) error {
	err := c.client.SendString(strMessage)
	if err != nil {
		return fmt.Errorf("failed sending message broadcast from node=%s id: %w", c.peer.me, err)
	}
	response, err := c.client.Receive()
	if err != nil {
		return fmt.Errorf("failed receiving message broadcast ACK on node=%s id: %w", c.peer.me, err)
	}
	if response != broadCastPeerResponse {
		fmt.Printf("Received unexpected response=%s on node=%s...\n", response, c.peer.me)
	}

	return nil
}

type broadcastResult struct {
	err      error
	nodeName string
}

// VectorClocksOpt ...
type VectorClocksOpt func(*VectorClocksPeer)

// WithTimeout ...
func WithTimeout(timeout time.Duration) VectorClocksOpt {
	return func(p *VectorClocksPeer) {
		p.timeout = &timeout
	}
}

// WithDebug ...
func WithDebug() VectorClocksOpt {
	return func(p *VectorClocksPeer) {
		// TODO: add debug logging
		p.isDebugOn = true
	}
}

// VectorClocksPeer ...
type VectorClocksPeer struct {
	me      topology.ServerInfo
	timeout *time.Duration
	server  *tcp.Server
	peers   map[string]*vectorClocksClient

	isDebugOn       bool
	sortedPeerNames []string
	messages        []VectorClocksMessage
	serverTimes     map[string]int
	serverTimeMU    *sync.Mutex
}

// NewVectorClocksPeer ...
func NewVectorClocksPeer(
	me topology.ServerInfo,
	peers []topology.ServerInfo,
	opts ...VectorClocksOpt) *VectorClocksPeer {

	p := &VectorClocksPeer{
		me:           me,
		peers:        make(map[string]*vectorClocksClient),
		serverTimes:  make(map[string]int),
		serverTimeMU: &sync.Mutex{},
	}
	for _, opt := range opts {
		opt(p)
	}

	p.initServer()
	p.initPeers(peers)

	return p
}

// Start ...
func (p *VectorClocksPeer) Start() error {
	err := p.server.Start(context.Background())
	if err != nil {
		return err
	}

	return nil
}

// Stop ...
func (lp *VectorClocksPeer) Stop() {
	defer lp.server.Stop()

	for _, p := range lp.peers {
		p.client.Close()
	}
}

// WhoAmI ...
func (p *VectorClocksPeer) WhoAmI() string {
	p.serverTimeMU.Lock()
	var times []string
	for nodeName, t := range p.serverTimes {
		times = append(times, fmt.Sprintf("%s:%d", nodeName, t))
	}
	p.serverTimeMU.Unlock()

	if p.isDebugOn {
		return fmt.Sprintf("[-->%s<-- %+v]", p.me.String(), times)
	}

	return p.me.String()
}

// Messages ...
func (p *VectorClocksPeer) Messages() []string {
	sort.Slice(p.messages, func(i, j int) bool {
		m1 := p.messages[i]
		m2 := p.messages[j]
		happensBeforeCount := 0
		for _, nodeName := range p.sortedPeerNames {
			if m1.Timestamps[nodeName] < m2.Timestamps[nodeName] {
				happensBeforeCount++
			}
		}

		return happensBeforeCount == len(p.sortedPeerNames)
	})

	var messages []string
	for _, m := range p.messages {
		msg := m.Body
		if p.isDebugOn {
			msg = fmt.Sprintf("%s:%+v", m.Body, m.Timestamps)
		}
		messages = append(messages, msg)
	}
	return messages
}

// Broadcast ...
func (p *VectorClocksPeer) Broadcast(message string) error {
	m := p.newMessage(message)
	bytes, err := Serialize(m)
	if err != nil {
		return fmt.Errorf("failed marshalling message=%s on node=%s timestamp: %w",
			message,
			p.me,
			err)
	}
	strMessage := string(bytes)

	wg := sync.WaitGroup{}
	wg.Add(len(p.peers))
	statusChan := make(chan broadcastResult, len(p.peers))

	for nodeName, p := range p.peers {
		go func(node string, client *vectorClocksClient) {
			defer wg.Done()

			err = client.sendBroadcast(strMessage)
			statusChan <- broadcastResult{
				err:      err,
				nodeName: node,
			}
		}(nodeName, p)
	}

	wg.Wait()

	var errs []error
	for i := 0; i < len(p.peers); i++ {
		status := <-statusChan
		if status.err != nil {
			errs = append(errs, status.err)
		}
	}
	return errors.Join(errs...)
}

func (p *VectorClocksPeer) receiveBroadcast(strMessage string) (bool, string, error) {
	fmt.Printf("Handling broadcast on node=%s....\n", p.me)
	fmt.Printf("Received in broadcast handler=%s on node=%s...\n", strMessage, p.me)

	var m VectorClocksMessage
	err := Deserialize([]byte(strMessage), &m)
	if err != nil {
		fmt.Printf("Unable to unmarshall Lamport Message on node=%s message=%s...\n", p.me, strMessage)
		return false, "", fmt.Errorf("unable to unmarshall json: %w", err)
	}

	p.serverTimeMU.Lock()

	// update local server times
	for node, ts := range m.Timestamps {
		if node == p.me.NodeName {
			continue
		}
		existingTs := p.serverTimes[node]
		ts := math.Max(float64(ts), float64(existingTs))
		p.serverTimes[node] = int(ts)
	}
	p.serverTimes[p.me.NodeName]++

	// add message to local messages
	for nodeName, ts := range p.serverTimes {
		m.Timestamps[nodeName] = ts
	}
	p.messages = append(p.messages, m)
	p.serverTimeMU.Unlock()

	fmt.Printf("Received message=%s on node=%s...\n", m.Body, p.WhoAmI())
	return false, broadCastPeerResponse, nil
}

func (p *VectorClocksPeer) initServer() {
	sb := tcp.NewStringStreamBuilder(p.receiveBroadcast)
	server := tcp.NewServer(
		p.me.PortNumber,
		tcp.WithStreamBuilder(sb),
		tcp.WithIP(p.me.IpAddress))
	p.server = server

	p.serverTimes[p.me.NodeName] = 0
}

func (p *VectorClocksPeer) initPeers(peers []topology.ServerInfo) {
	for _, peer := range peers {
		opts := []tcp.ClientOpt{tcp.WithClientIP(peer.IpAddress)}
		if p.timeout != nil {
			opts = append(opts, tcp.WithClientTimeout(*p.timeout))
		}

		c := tcp.NewClient(peer.PortNumber, opts...)
		p.peers[peer.NodeName] = newVectorClocksClient(peer, p, c)
		p.serverTimes[peer.NodeName] = 0
		p.sortedPeerNames = append(p.sortedPeerNames, peer.NodeName)
	}

	sort.Strings(p.sortedPeerNames)
}

func (p *VectorClocksPeer) newMessage(val string) VectorClocksMessage {
	p.serverTimeMU.Lock()
	defer p.serverTimeMU.Unlock()

	p.serverTimes[p.me.NodeName]++
	messageTimes := make(map[string]int)
	for nodeName, ts := range p.serverTimes {
		messageTimes[nodeName] = ts
	}
	m := VectorClocksMessage{
		Origin:     p.me,
		Timestamps: messageTimes,
		Body:       val,
	}
	p.messages = append(p.messages, m)

	return m
}
