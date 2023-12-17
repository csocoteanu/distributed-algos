package tcp

import (
	"bufio"
	"fmt"
)

const (
	BYTETYPE MultiStreamMessageType = 1 << iota
	INTTYPE
	STRINGTYPE
	INTSLICETYPE
	STRSLICETYPE
)

const MultiStreamMessageEndByte = 0xff

// MultiStreamMessageType ...
type MultiStreamMessageType uint8

// AddType ...
func (t *MultiStreamMessageType) AddType(t2 MultiStreamMessageType) {
	*t = *t | t2
}

// HasType ...
func (t MultiStreamMessageType) HasType(t2 MultiStreamMessageType) bool {
	return t&t2 != 0
}

// MultiStreamMessage ...
type MultiStreamMessage struct {
	Types       MultiStreamMessageType
	ByteValue   *byte
	IntValue    *int
	StringValue *string
	IntValues   []int
	StrValues   []string
}

// Read ...
func (msm *MultiStreamMessage) Read(reader *bufio.Reader) error {
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return err
		}

		if b == MultiStreamMessageEndByte {
			break
		}

		switch MultiStreamMessageType(b) {
		case BYTETYPE:
			readByte, err := reader.ReadByte()
			if err != nil {
				return fmt.Errorf("failed reading byte value: %w", err)
			}
			msm.ByteValue = &readByte
		case INTTYPE:
			intVal, err := readInt(reader)
			if err != nil {
				return fmt.Errorf("failed reading int value: %w", err)
			}
			msm.IntValue = &intVal
		case STRINGTYPE:
			strVal, err := readString(reader)
			if err != nil {
				return fmt.Errorf("failed reading string value: %w", err)
			}
			msm.StringValue = &strVal
		case INTSLICETYPE:
			sliceLen, err := readInt(reader)
			if err != nil {
				return fmt.Errorf("failed reading int slice length: %w", err)
			}
			for i := 0; i < sliceLen; i++ {
				intVal, err := readInt(reader)
				if err != nil {
					return fmt.Errorf("failed reading int slice element: %w", err)
				}
				msm.IntValues = append(msm.IntValues, intVal)
			}
		case STRSLICETYPE:
			sliceLen, err := readInt(reader)
			if err != nil {
				return fmt.Errorf("failed reading string slice length: %w", err)
			}
			for i := 0; i < sliceLen; i++ {
				strVal, err := readString(reader)
				if err != nil {
					return fmt.Errorf("failed reading string slice element: %w", err)
				}
				msm.StrValues = append(msm.StrValues, strVal)
			}
		default:
			return fmt.Errorf("unknown type: %v", b)
		}
	}

	return nil
}

// Send ...
func (msm *MultiStreamMessage) Send(c *Client) error {
	var err error

	if msm.ByteValue != nil {
		err = c.SendByte(byte(BYTETYPE))
		if err != nil {
			return fmt.Errorf("failed sending byte type: %w", err)
		}
		err = c.SendByte(*msm.ByteValue)
		if err != nil {
			return fmt.Errorf("failed sending byte value=%+v: %w", *msm.ByteValue, err)
		}
	}

	if msm.IntValue != nil {
		err = c.SendByte(byte(INTTYPE))
		if err != nil {
			return fmt.Errorf("failed sending int type: %w", err)
		}
		err = c.SentInt(*msm.IntValue)
		if err != nil {
			return fmt.Errorf("failed sending int value=%+v: %w", *msm.IntValue, err)
		}
	}

	if msm.StringValue != nil {
		err = c.SendByte(byte(STRINGTYPE))
		if err != nil {
			return fmt.Errorf("failed sending string type: %w", err)
		}
		err = c.SendString(*msm.StringValue)
		if err != nil {
			return fmt.Errorf("failed sending string value=%+v: %w", *msm.StringValue, err)
		}
	}

	if len(msm.IntValues) > 0 {
		err = c.SendByte(byte(INTSLICETYPE))
		if err != nil {
			return fmt.Errorf("failed sending int slice type: %w", err)
		}
		err = c.SendIntSlice(msm.IntValues)
		if err != nil {
			return fmt.Errorf("failed sending int slice value=%+v: %w", msm.IntValues, err)
		}
	}

	if len(msm.StrValues) > 0 {
		err = c.SendByte(byte(STRSLICETYPE))
		if err != nil {
			return fmt.Errorf("failed sending string slice type: %w", err)
		}
		err = c.SendStringSlice(msm.StrValues)
		if err != nil {
			return fmt.Errorf("failed sending string slice value=%+v: %w", msm.StrValues, err)
		}
	}

	return c.SendByte(MultiStreamMessageEndByte)
}

// MultiStreamReaderOpt ...
type MultiStreamReaderOpt func(*MultiStreamReader)

// MultiStreamMessageHandler ...
type MultiStreamMessageHandler struct {
	Identifier byte
	Handler    func(MultiStreamMessage) (bool, string, error)
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

	msm := MultiStreamMessage{}
	err = msm.Read(reader)
	if err != nil {
		return false, "", fmt.Errorf("failed reading multistream message: %w", err)
	}

	return h.Handler(msm)
}
