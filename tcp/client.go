package tcp

import (
	"fmt"
	"net"
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
}

// NewClient ...
func NewClient(port int, opts ...ClientOpt) (*Client, error) {
	c := Client{
		port: port,
	}
	for _, opt := range opts {
		opt(&c)
	}

	remoteAddr := fmt.Sprintf("%s:%d", c.ip, c.port)
	conn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		return nil, err
	}

	c.conn = conn
	if c.buffSize == 0 {
		c.buffSize = defaultBuffSize
	}

	return &c, nil
}

// Send ...
func (c *Client) Send(data string) error {
	raw := []byte(data)

	return c.SendRaw(raw)
}

// SendRaw ...
func (c *Client) SendRaw(data []byte) error {
	_, err := c.conn.Write(data)
	if err != nil {
		fmt.Printf("Error sending data: %v\n", err)
		return err
	}

	return nil
}

// Receive ...
func (c *Client) Receive() (string, error) {
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
	return c.conn.Close()
}
