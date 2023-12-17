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

// StringHandler ...
type StringHandler func(string) (bool, string, error)

// StringStreamBuilder ...
type StringStreamBuilder struct {
	fn StringHandler
}

// NewStringStreamBuilder ...
func NewStringStreamBuilder(fn StringHandler) *StringStreamBuilder {
	return &StringStreamBuilder{
		fn: fn,
	}
}

// NewStringStreamBuilder ...
func (ssb *StringStreamBuilder) Build() (Stream, error) {
	return &StringReader{
		handler: ssb.fn,
	}, nil
}

// StringReader ...
type StringReader struct {
	handler func(string) (bool, string, error)
}

// ReadNext ...
func (sr *StringReader) ReadNext(reader *bufio.Reader) (bool, string, error) {
	str, err := readString(reader)
	if err != nil {
		return false, "", fmt.Errorf("error reading string: %w", err)
	}

	result := str
	if sr.handler != nil {
		return sr.handler(result)
	}

	return false, result, nil
}

func readInt(reader *bufio.Reader) (int, error) {
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

func readString(reader *bufio.Reader) (string, error) {
	len, err := readInt(reader)
	if err != nil {
		return "", fmt.Errorf("failed reading string length: %w", err)
	}

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
