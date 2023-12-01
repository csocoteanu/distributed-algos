package algos

import (
	"bufio"
	"context"
	"distributed-algos/tcp"
	"fmt"
	"io"
)

// LamportPeer
type LamportPeer struct {
	server  *tcp.Server
	clients []*tcp.Client
}

// NewLamportPeer ...
func NewLamportPeer(port int, opts ...tcp.ServerOpt) (*LamportPeer, error) {
	eh := EchoHandler{}
	sb := tcp.NewMultiStreamBuilder(
		tcp.WithMultiStreamHandler(tcp.MultiStreamMessageHandler{
			Identifier: 'e',
			Handler:    eh.Handle,
		}),
	)

	server := tcp.NewServer(port, tcp.WithStreamBuilder(sb))

	return &LamportPeer{
		server: server,
	}, nil
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

// EchoHandler ...
type EchoHandler struct {
}

// Handle ...
func (eh *EchoHandler) Handle(len uint8, reader *bufio.Reader) (bool, string, error) {
	fmt.Println("Handling echo....")

	var allBytes []byte
	count := uint8(0)
	for {
		b, err := reader.ReadByte()
		fmt.Println("Reading byte..." + string(b))
		if err == io.EOF {
			break
		}

		allBytes = append(allBytes, b)
		count++
		if count == len {
			break
		}
	}

	fmt.Printf("Received: %s\n", string(allBytes))

	return false, string(allBytes), nil
}

type PingPongHandler struct {
}

type LamportClient struct {
	client *tcp.Client
}

func NewLamportClient(port int) (*LamportClient, error) {
	c, err := tcp.NewClient(port)
	if err != nil {
		return nil, err
	}

	return &LamportClient{
		client: c,
	}, nil
}

func (lc *LamportClient) Send(data string) error {
	err := lc.client.Send("e")
	if err != nil {
		return err
	}

	len := uint8(len(data))
	err = lc.client.SendRaw([]byte{len})
	if err != nil {
		return err
	}

	return lc.client.Send(data)
}

func (lc *LamportClient) Close() error {
	return lc.client.Close()
}

func (lc *LamportClient) Receive() (string, error) {
	return lc.client.Receive()
}
