package tcp

import (
	"fmt"
	"net"
	"sync"
	"time"
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

// WithClientTimeout ...
func WithClientTimeout(timeout time.Duration) ClientOpt {
	return func(c *Client) {
		c.timeout = &timeout
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
	timeout  *time.Duration
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

// SentInt ...
func (c *Client) SentInt(data int) error {
	err := c.initConnection()
	if err != nil {
		return fmt.Errorf("failed initializing client connection: %w", err)
	}

	raw := []byte{
		byte(data >> 24),
		byte(data >> 16),
		byte(data >> 8),
		byte(data),
	}
	return c.sendBytes(raw)
}

// SendString ...
func (c *Client) SendString(str string) error {
	err := c.initConnection()
	if err != nil {
		return fmt.Errorf("failed initializing client connection: %w", err)
	}

	strLen := len(str)
	err = c.SentInt(strLen)
	if err != nil {
		return fmt.Errorf("failed sending string length: %w", err)
	}

	raw := []byte(str)
	return c.sendBytes(raw)
}

// SendIntSlice ...
func (c *Client) SendIntSlice(ints []int) error {
	err := c.initConnection()
	if err != nil {
		return fmt.Errorf("failed initializing client connection: %w", err)
	}

	err = c.SentInt(len(ints))
	if err != nil {
		return fmt.Errorf("failed sending int slice length: %w", err)
	}

	for _, i := range ints {
		err = c.SentInt(i)
		if err != nil {
			return fmt.Errorf("failed sending int slice element: %w", err)
		}
	}

	return nil
}

// SendStringSlice ...
func (c *Client) SendStringSlice(strings []string) error {
	err := c.initConnection()
	if err != nil {
		return fmt.Errorf("failed initializing client connection: %w", err)
	}

	err = c.SentInt(len(strings))
	if err != nil {
		return fmt.Errorf("failed sending string slice length: %w", err)
	}

	for _, s := range strings {
		err = c.SendString(s)
		if err != nil {
			return fmt.Errorf("failed sending string slice element: %w", err)
		}
	}

	return nil
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
	var conn net.Conn
	var err error
	if c.timeout != nil {
		d := net.Dialer{Timeout: *c.timeout}
		conn, err = d.Dial("tcp", remoteAddr)
	} else {
		conn, err = net.Dial("tcp", remoteAddr)
	}
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}
