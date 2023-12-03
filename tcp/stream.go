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

// MultiStreamReaderOpt ...
type MultiStreamReaderOpt func(*MultiStreamReader)

// MultiStreamMessageHandler ...
type MultiStreamMessageHandler struct {
	Identifier byte
	Handler    func(string) (bool, string, error)
}

// WithMultiStreamHandler ...
func WithMultiStreamHandler(handler MultiStreamMessageHandler) MultiStreamReaderOpt {
	return func(msr *MultiStreamReader) {
		msr.handlers[handler.Identifier] = handler
	}
}

// MultiStreamBuilder ...
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

	len, err := readStringLength(reader)
	if err != nil {
		fmt.Printf("Error reading length bytes: %v\n", err)
		return false, "", err
	}

	str, err := readString(reader, len)
	if err != nil {
		fmt.Printf("Error reading string bytes: %v\n", err)
	}

	return h.Handler(str)
}

func readStringLength(reader *bufio.Reader) (int, error) {
	len := 0
	for i := 0; i < 4; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}

		len = len << 8
		len = len | int(b)
	}

	return len, nil
}

func readString(reader *bufio.Reader, len int) (string, error) {
	var allBytes []byte
	count := 0
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}

		allBytes = append(allBytes, b)
		count++
		if count == len {
			break
		}
	}

	return string(allBytes), nil
}
