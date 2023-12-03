package tcp

import (
	"bufio"
	"context"
	"fmt"
	"net"
)

// ServerOpt ...
type ServerOpt func(*Server)

// Server ...
type Server struct {
	port          int
	ipAddress     string
	streamBuilder StreamBuilder

	tcpContext    context.Context
	tcpCancelFunc context.CancelFunc
	tcpListener   net.Listener
}

// WithIP ...
func WithIP(ipAddress string) ServerOpt {
	return func(s *Server) {
		s.ipAddress = ipAddress
	}
}

// WithStreamBuilder ...
func WithStreamBuilder(b StreamBuilder) ServerOpt {
	return func(s *Server) {
		s.streamBuilder = b
	}
}

// NewServer ...
func NewServer(port int, opts ...ServerOpt) *Server {
	s := Server{
		port: port,
	}

	for _, opt := range opts {
		opt(&s)
	}

	return &s
}

// Start ...
func (s *Server) Start(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", s.ipAddress, s.port)
	tcpListener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.acceptConnections(ctx, tcpListener)

	return nil
}

// Stop ...
func (s *Server) Stop() {
	if s.tcpCancelFunc != nil {
		s.tcpCancelFunc()
	}
}

func (s *Server) acceptConnections(ctx context.Context, tcpListener net.Listener) {
	s.tcpListener = tcpListener
	s.tcpContext, s.tcpCancelFunc = context.WithCancel(ctx)

	defer s.tcpListener.Close()

	for {
		select {
		case <-s.tcpContext.Done():
			return
		default:
			connection, err := s.tcpListener.Accept()
			if err != nil {
				fmt.Printf("Could not accept connections: %v\n", err)
				return
			}
			go s.handleConnection(s.tcpContext, connection)
		}
	}
}

func (s *Server) handleConnection(ctx context.Context, c net.Conn) {
	defer c.Close()

	remoteAddr := c.RemoteAddr().String()
	fmt.Printf("Serving %s\n", remoteAddr)

	if s.streamBuilder == nil {
		return
	}
	stream, err := s.streamBuilder.Build()
	if err != nil {
		fmt.Printf("Could not build stream for %s: %v\n", remoteAddr, err)
		return
	}

	for {
		r := bufio.NewReader(c)
		hasNext, response, err := stream.ReadNext(r)
		if err != nil {
			fmt.Printf("Could not read next from stream: %v\n", err)
			return
		}

		if len(response) > 0 {
			fmt.Println("Writing response from Server..." + response)
			_, err = c.Write([]byte(response))
			if err != nil {
				fmt.Printf("Could not write to connection response: %v\n", err)
				return
			}
		}

		if !hasNext {
			break
		}
	}
}
