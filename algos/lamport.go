package algos

import (
	"context"
	"distributed-algos/tcp"
	"fmt"
)

const (
	EchoMessageIdentifier     = 'e'
	PingPongMessageIdentifier = 'p'
)

// LamportPeer
type LamportPeer struct {
	server  *tcp.Server
	clients []*tcp.Client
}

// NewLamportPeer ...
func NewLamportPeer(port int, opts ...tcp.ServerOpt) *LamportPeer {
	lp := &LamportPeer{}

	sb := tcp.NewMultiStreamBuilder(
		tcp.WithMultiStreamHandler(tcp.MultiStreamMessageHandler{
			Identifier: 'e',
			Handler:    lp.HandleEcho,
		}),
		tcp.WithMultiStreamHandler(tcp.MultiStreamMessageHandler{
			Identifier: 'p',
			Handler:    lp.HandlePingPong,
		}),
	)

	server := tcp.NewServer(port, tcp.WithStreamBuilder(sb))
	lp.server = server

	return lp
}

func (lp *LamportPeer) Start() error {
	err := lp.server.Start(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (lp *LamportPeer) Stop() {
	lp.server.Stop()
}

// HandleEcho ...
func (lp *LamportPeer) HandleEcho(str string) (bool, string, error) {
	fmt.Println("Handling echo....")
	fmt.Printf("Received in echo handler: %s\n", str)

	return false, str, nil
}

// HandlePingPong ...
func (lp *LamportPeer) HandlePingPong(str string) (bool, string, error) {
	fmt.Println("Handling ping pong....")
	fmt.Printf("Received in ping pong handler: %s\n", str)

	if str != "ping" {
		return false, "", fmt.Errorf("expected ping, got %s", str)
	}

	return false, "pong", nil
}

// LamportClient ...
type LamportClient struct {
	client *tcp.Client
}

// NewLamportClient ...
func NewLamportClient(port int) *LamportClient {
	c := tcp.NewClient(port)
	return &LamportClient{
		client: c,
	}
}

// Send ...
func (lc *LamportClient) Send(id byte, data string) error {
	switch id {
	case EchoMessageIdentifier, PingPongMessageIdentifier:
		break
	default:
		return fmt.Errorf("invalid message identifier: %d", id)
	}

	err := lc.client.SendByte(id)
	if err != nil {
		return err
	}

	return lc.client.SendString(data)
}

// Close ...
func (lc *LamportClient) Close() error {
	return lc.client.Close()
}

// Receive ...
func (lc *LamportClient) Receive() (string, error) {
	return lc.client.Receive()
}
