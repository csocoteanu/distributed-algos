package algos

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
	broadcastMessageType  = 'b'
	broadCastPeerResponse = "ACK"
)

// LamportMessage ...
type LamportMessage struct {
	Origin    topology.ServerInfo `json:"origin"`
	Body      string              `json:"body"`
	Timestamp int                 `json:"timestamp"`
}

// Serialize ...
func Serialize(m LamportMessage) ([]byte, error) {
	return json.Marshal(m)
}

// Deserialize ...
func Deserialize(b []byte, m *LamportMessage) error {
	if m == nil {
		return fmt.Errorf("unexpected nil pointer received")
	}

	return json.Unmarshal(b, m)
}

type lamportClient struct {
	nodeInfo topology.ServerInfo
	peer     *LamportPeer
	client   *tcp.Client
}

func newLamportClient(
	nodeInfo topology.ServerInfo,
	peer *LamportPeer,
	client *tcp.Client) *lamportClient {

	return &lamportClient{
		nodeInfo: nodeInfo,
		peer:     peer,
		client:   client,
	}
}

func (lc *lamportClient) sendBroadcast(strMessage string) error {
	err := lc.client.SendByte(broadcastMessageType)
	if err != nil {
		return fmt.Errorf("failed sending message id broadcast from node=%s id: %w", lc.peer.me, err)
	}
	err = lc.client.SendString(strMessage)
	if err != nil {
		return fmt.Errorf("failed sending message broadcast from node=%s id: %w", lc.peer.me, err)
	}
	response, err := lc.client.Receive()
	if err != nil {
		return fmt.Errorf("failed receiving message broadcast ACK on node=%s id: %w", lc.peer.me, err)
	}
	if response != "ACK" {
		fmt.Printf("Received unexpected response=%s on node=%s...\n", response, lc.peer.me)
	}

	return nil
}

type broadcastResult struct {
	err      error
	nodeName string
}

// LamportOpt ...
type LamportOpt func(*LamportPeer)

// WithLamportTimeout ...
func WithLamportTimeout(timeout time.Duration) LamportOpt {
	return func(lp *LamportPeer) {
		lp.timeout = &timeout
	}
}

// LamportPeer ...
type LamportPeer struct {
	me           topology.ServerInfo
	peers        map[string]*lamportClient
	messages     []LamportMessage
	timeout      *time.Duration
	serverTime   int
	serverTimeMU *sync.Mutex

	server *tcp.Server
}

// NewLamportPeer ...
func NewLamportPeer(
	me topology.ServerInfo,
	peers []topology.ServerInfo,
	opts ...LamportOpt) *LamportPeer {

	lp := &LamportPeer{
		me:           me,
		peers:        make(map[string]*lamportClient),
		serverTime:   0,
		serverTimeMU: &sync.Mutex{},
	}
	for _, opt := range opts {
		opt(lp)
	}

	lp.initServer()
	lp.initPeers(peers)

	return lp
}

// Start ...
func (lp *LamportPeer) Start() error {
	err := lp.server.Start(context.Background())
	if err != nil {
		return err
	}

	return nil
}

// Stop ...
func (lp *LamportPeer) Stop() {
	defer lp.server.Stop()

	for _, p := range lp.peers {
		p.client.Close()
	}
}

// WhoAmI ...
func (lp *LamportPeer) WhoAmI() string {
	lp.serverTimeMU.Lock()
	ts := lp.serverTime
	lp.serverTimeMU.Unlock()

	return fmt.Sprintf("[%s - %d]", lp.me.String(), ts)
}

// Messages ...
func (lp *LamportPeer) Messages() []LamportMessage {
	sort.Slice(lp.messages, func(i, j int) bool {
		return lp.messages[i].Timestamp <= lp.messages[j].Timestamp
	})

	return lp.messages
}

// Broadcast ...
// TODO: should we rollback the server time in case of failures?
func (lp *LamportPeer) Broadcast(message string) error {
	lp.serverTimeMU.Lock()
	lp.serverTime++
	m := LamportMessage{
		Origin:    lp.me,
		Body:      message,
		Timestamp: lp.serverTime,
	}
	lp.messages = append(lp.messages, m)
	lp.serverTimeMU.Unlock()

	bytes, err := Serialize(m)
	if err != nil {
		return fmt.Errorf("failed marshalling message=%s on node=%s timestamp: %w",
			message,
			lp.me,
			err)
	}
	strMessage := string(bytes)

	wg := sync.WaitGroup{}
	wg.Add(len(lp.peers))
	statusChan := make(chan broadcastResult, len(lp.peers))

	for _, p := range lp.peers {
		go func(client *lamportClient) {
			defer wg.Done()

			err = client.sendBroadcast(strMessage)
			statusChan <- broadcastResult{
				err:      err,
				nodeName: lp.me.String(),
			}
		}(p)
	}

	wg.Wait()

	var errs []error
	for i := 0; i < len(lp.peers); i++ {
		status := <-statusChan
		if status.err != nil {
			errs = append(errs, status.err)
		}
	}
	return errors.Join(errs...)
}

func (lp *LamportPeer) initServer() {
	sb := tcp.NewMultiStreamBuilder(
		tcp.WithMultiStreamHandler(tcp.MultiStreamMessageHandler{
			Identifier: broadcastMessageType,
			Handler:    lp.receiveBroadcast,
		}),
	)
	server := tcp.NewServer(
		lp.me.PortNumber,
		tcp.WithStreamBuilder(sb),
		tcp.WithIP(lp.me.IpAddress))
	lp.server = server
}

func (lp *LamportPeer) initPeers(peers []topology.ServerInfo) {

	for _, p := range peers {
		opts := []tcp.ClientOpt{tcp.WithClientIP(p.IpAddress)}
		if lp.timeout != nil {
			opts = append(opts, tcp.WithClientTimeout(*lp.timeout))
		}

		c := tcp.NewClient(p.PortNumber, opts...)
		lp.peers[p.NodeName] = newLamportClient(p, lp, c)
	}
}

func (lp *LamportPeer) receiveBroadcast(str string) (bool, string, error) {
	fmt.Printf("Handling broadcast on node=%s....\n", lp.me)
	fmt.Printf("Received in broadcast handler=%s on node=%s...\n", str, lp.me)

	var m LamportMessage
	err := Deserialize([]byte(str), &m)
	if err != nil {
		fmt.Printf("Unable to unmarshall Lamport Message on node=%s message=%s...\n",
			lp.me,
			str)
		return false, "", fmt.Errorf("unable to unmarshall json: %w", err)
	}

	lp.serverTimeMU.Lock()
	lp.serverTime++
	ts := math.Max(float64(m.Timestamp), float64(lp.serverTime)) + 1
	lp.serverTime = int(ts)
	lp.messages = append(lp.messages, m)
	lp.serverTimeMU.Unlock()
	fmt.Printf("Received message=%s on node=%s and now time is=%d...\n", m.Body, lp.me, int(ts))

	return false, broadCastPeerResponse, nil
}
