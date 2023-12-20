package tcp

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// WaitForServerStart ...
func WaitForServerStart(ctx context.Context, server *Server) error {
	isStarted := false
	startTimer := time.NewTimer(200 * time.Millisecond)
	cancelCtx, cancelFunc := context.WithTimeout(ctx, 10*time.Second)

	defer startTimer.Stop()
	defer cancelFunc()

	for !isStarted {
		select {
		case <-startTimer.C:
			isStarted = server.IsStarted()
			if isStarted {
				return nil
			}
		case <-cancelCtx.Done():
			if !server.IsStarted() {
				return fmt.Errorf("tcp server failed starting in grace period...")
			}
		}
	}

	return nil
}

func newTestServer(portNum int, callback StringHandler) *Server {
	sb := NewStringStreamBuilder(callback)

	return NewServer(portNum, WithStreamBuilder(sb))
}

func handleEcho(str string) (bool, string, error) {
	fmt.Println("Handling echo....")
	fmt.Printf("Received in echo handler: %s\n", str)

	return false, str, nil
}

func handlePingPong(str string) (bool, string, error) {
	fmt.Println("Handling ping pong....")
	fmt.Printf("Received in ping pong handler: %s\n", str)

	if str != "ping" {
		return false, "", fmt.Errorf("expected ping, got %s", str)
	}

	return false, "pong", nil
}

type tcpTestCase struct {
	tcName        string
	portNum       int
	fn            StringHandler
	inputMessage  string
	outputMessage string
	hasError      bool
}

func TestStringStreamReadNext(t *testing.T) {
	testCases := []tcpTestCase{
		{
			tcName:        "echo",
			portNum:       9090,
			fn:            handleEcho,
			inputMessage:  "Hello world!",
			outputMessage: "Hello world!",
			hasError:      false,
		},
		{
			tcName:        "ping-pong",
			portNum:       9091,
			fn:            handlePingPong,
			inputMessage:  "ping",
			outputMessage: "pong",
			hasError:      false,
		},
		{
			tcName:        "ping-pong with invalid message type",
			portNum:       9092,
			fn:            handlePingPong,
			inputMessage:  "Hello!",
			outputMessage: "",
			hasError:      true,
		},
	}

	for _, tc := range testCases {
		fmt.Printf("Running %s...\n", tc.tcName)

		server := newTestServer(tc.portNum, tc.fn)
		ctx := context.Background()
		go func() {
			err := server.Start(ctx)
			if err != nil {
				fmt.Printf("failed starting %s server: %v", tc.tcName, err)
			}
		}()
		defer server.Stop()

		err := WaitForServerStart(ctx, server)
		if err != nil {
			t.Fatalf("failed starting %s server: %v", tc.tcName, err)
		}

		tcpClient := NewClient(tc.portNum)
		defer tcpClient.Close()

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
