package tcp

import (
	"fmt"
	"net"
	"sync"
)

const defaultBuffSize = 1024

// ClientOpt ...
type ClientOpt func(*Client)

// WithClientIP ...
func WithClientIP(ip string) ClientOpt {
	return func(c *Client) {
		c.ip = ip
	}
}

// WithBuffSize ...
func WithBuffSize(size int) ClientOpt {
	return func(c *Client) {
		c.buffSize = size
	}
}

// Client ...
type Client struct {
	port     int
	ip       string
	buffSize int
	conn     net.Conn
	connMu   *sync.Mutex
}

// NewClient ...
func NewClient(port int, opts ...ClientOpt) *Client {
	c := Client{
		port:   port,
		connMu: &sync.Mutex{},
	}
	for _, opt := range opts {
		opt(&c)
	}

	if c.buffSize == 0 {
		c.buffSize = defaultBuffSize
	}

	return &c
}

// SendByte ...
func (c *Client) SendByte(data byte) error {
	err := c.initConnection()
	if err != nil {
		return fmt.Errorf("failed initializing client connection: %w", err)
	}

	return c.sendBytes([]byte{data})
}

// SendString ...
func (c *Client) SendString(str string) error {
	err := c.initConnection()
	if err != nil {
		return fmt.Errorf("failed initializing client connection: %w", err)
	}

	strLen := len(str)
	raw := []byte{
		byte(strLen >> 24),
		byte(strLen >> 16),
		byte(strLen >> 8),
		byte(strLen),
	}
	err = c.sendBytes(raw)
	if err != nil {
		fmt.Printf("Error sending length=%d for string=%s: %v\n",
			strLen,
			str,
			err)
		return fmt.Errorf("failed sending bytes: %w", err)
	}

	raw = []byte(str)
	return c.sendBytes(raw)
}

func (c *Client) sendBytes(data []byte) error {
	_, err := c.conn.Write(data)
	if err != nil {
		fmt.Printf("Error sending data: %v\n", err)
		return err
	}

	return nil
}

// Receive ...
func (c *Client) Receive() (string, error) {
	defer c.Close()

	data := make([]byte, c.buffSize)
	var result string

	for {
		n, err := c.conn.Read(data)
		if err != nil {
			return "", err
		}

		result += string(data[:n])
		if n < c.buffSize {
			break
		}
	}

	return result, nil
}

// Close ...
func (c *Client) Close() error {
	c.connMu.Lock()
	defer func() {
		c.conn = nil
		c.connMu.Unlock()
	}()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) initConnection() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		return nil
	}

	remoteAddr := fmt.Sprintf("%s:%d", c.ip, c.port)
	conn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}
