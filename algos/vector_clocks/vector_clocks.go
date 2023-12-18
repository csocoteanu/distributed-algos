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

// VectorClocksPeer ...
type VectorClocksPeer struct {
	me           topology.ServerInfo
	peers        map[string]*vectorClocksClient
	messages     []VectorClocksMessage
	timeout      *time.Duration
	serverTimes  map[string]int
	serverTimeMU *sync.Mutex

	server *tcp.Server
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
	var times []int
	for _, t := range p.serverTimes {
		times = append(times, t)
	}
	p.serverTimeMU.Unlock()

	return fmt.Sprintf("[%s - %+v]", p.me.String(), times)
}

// Messages ...
func (p *VectorClocksPeer) Messages() []VectorClocksMessage {
	sort.Slice(p.messages, func(i, j int) bool {
		nodeI := p.messages[i].Origin.NodeName
		nodeJ := p.messages[j].Origin.NodeName
		return p.serverTimes[nodeI] <= p.serverTimes[nodeJ]
	})

	return p.messages
}

// Broadcast ...
func (p *VectorClocksPeer) Broadcast(message string) error {
	p.serverTimeMU.Lock()
	p.serverTimes[p.me.NodeName]++
	m := p.newMessage(message)
	p.messages = append(p.messages, m)
	p.serverTimeMU.Unlock()

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
	}
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
	p.serverTimes[p.me.NodeName]++

	for node, ts := range m.Timestamps {
		existingTs := p.serverTimes[node]
		ts := math.Max(float64(ts), float64(existingTs)) + 1
		p.serverTimes[node] = int(ts)
	}

	p.messages = append(p.messages, m)
	p.serverTimeMU.Unlock()
	fmt.Printf("Received message=%s on node=%s...\n", m.Body, p.WhoAmI())

	return false, broadCastPeerResponse, nil
}

func (p *VectorClocksPeer) newMessage(val string) VectorClocksMessage {
	return VectorClocksMessage{
		Origin:     p.me,
		Timestamps: p.serverTimes,
		Body:       val,
	}
}
