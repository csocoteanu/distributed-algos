package tcp

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func newMultiStreamServer(portNum int) *Server {
	sb := NewMultiStreamBuilder(
		WithMultiStreamHandler(MultiStreamMessageHandler{
			Identifier: 1,
			Handler:    handleInts,
		}),
		WithMultiStreamHandler(MultiStreamMessageHandler{
			Identifier: 2,
			Handler:    handleStrings,
		}),
	)

	return NewServer(portNum, WithStreamBuilder(sb))
}

func handleInts(msg MultiStreamMessage) (bool, string, error) {
	fmt.Println("Handling ints....")
	fmt.Printf("Received in ints handler: %v\n", msg.IntValues)

	if len(msg.IntValues) == 0 {
		return false, "", fmt.Errorf("expected at least one int value")
	}

	sum := 0
	for _, i := range msg.IntValues {
		sum += i
	}
	return false, fmt.Sprintf("%d", sum), nil
}

func handleStrings(msg MultiStreamMessage) (bool, string, error) {
	fmt.Println("Handling strings....")
	fmt.Printf("Received in strings handler: %v\n", msg.StrValues)

	if len(msg.StrValues) == 0 {
		return false, "", fmt.Errorf("expected at least one string value")

	}

	return false, strings.Join(msg.StrValues, ","), nil
}

type tcpMultiStreamTestCase struct {
	tcName            string
	streamMessageType byte
	intValues         []int
	strValues         []string
	expectedOutput    string
}

func TestMultiStreamTCP(t *testing.T) {
	ctx := context.Background()
	server := newMultiStreamServer(10000)
	defer server.Stop()

	go func() {
		err := server.Start(ctx)
		if err != nil {
			fmt.Printf("failed starting server: %v", err)
		}
	}()
	err := waitForServerStart(ctx, server)
	if err != nil {
		t.Fatalf("failed waiting for server to start: %v", err)
	}

	client := NewClient(10000)
	defer client.Close()

	tests := []tcpMultiStreamTestCase{
		{
			tcName:            "testing multistream ints",
			streamMessageType: 1,
			intValues:         []int{1, 2, 3, 4, 5},
			expectedOutput:    "15",
		},
		{
			tcName:            "testing multistream strings",
			streamMessageType: 2,
			strValues:         []string{"hello", "world", "this", "is", "a", "test"},
			expectedOutput:    "hello,world,this,is,a,test",
		},
	}

	for _, tc := range tests {
		fmt.Printf("Running %s...\n", tc.tcName)

		msm := MultiStreamMessage{}
		if len(tc.intValues) > 0 {
			msm.IntValues = tc.intValues
		}
		if len(tc.strValues) > 0 {
			msm.StrValues = tc.strValues
		}

		err := client.SendByte(tc.streamMessageType)
		if err != nil {
			t.Fatalf("failed sending multistream message type=%+v: %v", tc.streamMessageType, err)
		}
		err = msm.Send(client)
		if err != nil {
			t.Fatalf("failed sending multistream message= %+v: %v", msm, err)
		}
		result, err := client.Receive()
		if err != nil {
			t.Fatalf("failed receiving multistream message: %v", err)
		}

		if result != tc.expectedOutput {
			t.Fatalf("expected output=%s, got=%s", tc.expectedOutput, result)
		}
	}
}
