package tcp

import (
	"bufio"
	"fmt"
)

// StreamBuilder ...
type StreamBuilder interface {
	Build() (Stream, error)
}

// Stream ...
type Stream interface {
	ReadNext(reader *bufio.Reader) (bool, string, error)
}

// SimpleStreamBuilder ...
type SimpleStreamBuilder struct {
}

// NewSimpleStreamBuilder ...
func NewSimpleStreamBuilder() *SimpleStreamBuilder {
	return &SimpleStreamBuilder{}
}

// Build ...
func (ssb *SimpleStreamBuilder) Build() (Stream, error) {
	return NewSimpleStreamReader(), nil
}

// SimpleStreamReader ...
type SimpleStreamReader struct {
}

// NewSimpleStreamReader ...
func NewSimpleStreamReader() *SimpleStreamReader {
	return &SimpleStreamReader{}
}

// ReadNext ...
func (ssr *SimpleStreamReader) ReadNext(reader *bufio.Reader) (bool, string, error) {
	data, err := reader.ReadString('\n')
	if err != nil {
		return false, "", err
	}

	fmt.Printf("Received: %s\n", string(data))

	return false, "ACK", nil
}

// MultiStreamReaderOpt ...
type MultiStreamReaderOpt func(*MultiStreamReader)

// WithMultiStreamHandler ...
func WithMultiStreamHandler(handler MultiStreamMessageHandler) MultiStreamReaderOpt {
	return func(msr *MultiStreamReader) {
		msr.handlers[handler.Identifier] = handler
	}
}

type MultiStreamBuilder struct {
	opts []MultiStreamReaderOpt
}

// NewMultiStreamBuilder ...
func NewMultiStreamBuilder(opts ...MultiStreamReaderOpt) *MultiStreamBuilder {
	return &MultiStreamBuilder{
		opts: opts,
	}
}

// Build ...
func (msb *MultiStreamBuilder) Build() (Stream, error) {
	msr := &MultiStreamReader{
		handlers: make(map[byte]MultiStreamMessageHandler),
	}

	for _, opt := range msb.opts {
		opt(msr)
	}

	return msr, nil
}

// MultiStreamReader ...
type MultiStreamReader struct {
	handlers map[byte]MultiStreamMessageHandler
}

// ReadNext ...
func (msr *MultiStreamReader) ReadNext(reader *bufio.Reader) (bool, string, error) {
	fmt.Println("Reading in multistream...")

	b, err := reader.ReadByte()
	if err != nil {
		fmt.Println("Error reading first byte")
		return false, "", err
	}
	h, ok := msr.handlers[b]
	if !ok {
		fmt.Println("No handler found in multistream...")
		return false, "", nil
	}

	len, err := reader.ReadByte()
	if err != nil {
		fmt.Println("Error reading length byte")
		return false, "", err
	}

	return h.Handler(len, reader)
}

type MultiStreamMessageHandler struct {
	Identifier byte
	Handler    func(len uint8, reader *bufio.Reader) (bool, string, error)
}
