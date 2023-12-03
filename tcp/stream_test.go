package tcp

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type testTCPPeer struct {
	server *Server
}

func newTestTCPPeer() *testTCPPeer {
	tp := testTCPPeer{}

	sb := NewMultiStreamBuilder(
		WithMultiStreamHandler(MultiStreamMessageHandler{
			Identifier: 'e',
			Handler:    tp.handleEcho,
		}),
		WithMultiStreamHandler(MultiStreamMessageHandler{
			Identifier: 'p',
			Handler:    tp.handlePingPong,
		}),
	)

	server := NewServer(19999, WithStreamBuilder(sb))
	tp.server = server

	return &tp
}

func (tp *testTCPPeer) handleEcho(str string) (bool, string, error) {
	fmt.Println("Handling echo....")
	fmt.Printf("Received in echo handler: %s\n", str)

	return false, str, nil
}

func (tp *testTCPPeer) handlePingPong(str string) (bool, string, error) {
	fmt.Println("Handling ping pong....")
	fmt.Printf("Received in ping pong handler: %s\n", str)

	if str != "ping" {
		return false, "", fmt.Errorf("expected ping, got %s", str)
	}

	return false, "pong", nil
}

type tcpTestCase struct {
	tcName        string
	messageType   byte
	inputMessage  string
	outputMessage string
	hasError      bool
}

func TestMultiStreamReader_ReadNext(t *testing.T) {
	tp := newTestTCPPeer()

	isStarted := false
	startTimer := time.NewTimer(200 * time.Millisecond)
	ctx, timeoutFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeoutFunc()

	go func() {
		err := tp.server.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()
	for !isStarted {
		select {
		case <-startTimer.C:
			isStarted = tp.server.IsStarted()
		case <-ctx.Done():
			if tp.server.IsStarted() {
				t.Fatal("tcp server failed starting in grace period...")
			}
		}
	}
	defer tp.server.Stop()

	tcpClient := NewClient(tp.server.port)
	defer tcpClient.Close()

	testCases := []tcpTestCase{
		{
			tcName:        "echo",
			messageType:   'e',
			inputMessage:  "Hello world!",
			outputMessage: "Hello world!",
			hasError:      false,
		},
		{
			tcName:        "ping-pong",
			messageType:   'p',
			inputMessage:  "ping",
			outputMessage: "pong",
			hasError:      false,
		},
		{
			tcName:        "ping-pong with invalid message type",
			messageType:   'p',
			inputMessage:  "Hello!",
			outputMessage: "",
			hasError:      true,
		},
	}

	for _, tc := range testCases {
		fmt.Printf("Running %s...\n", tc.tcName)

		err := tcpClient.SendByte(tc.messageType)
		if err != nil {
			t.Fatalf("unable to send %s message type: %v", tc.tcName, err)
		}

		err = tcpClient.SendString(tc.inputMessage)
		if err != nil {
			t.Fatalf("unable to send %s message string: %v", tc.tcName, err)
		}

		response, err := tcpClient.Receive()
		if tc.hasError {
			if err == nil {
				t.Fatalf("expected %s to fail when receiving message", tc.tcName)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unable to receive %s message string: %v", tc.tcName, err)
		}
		if response != tc.outputMessage {
			t.Fatalf("received unexpected %s message: %s - expected %s",
				tc.tcName,
				response,
				tc.outputMessage)
		}
	}
}
