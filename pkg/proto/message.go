package proto

import (
	"bytes"
	"fmt"
	"io"
)

var (
	MESSAGE_TERMINATOR = []byte{'\n', '\r'}
)

type Message struct {
	Command string
	Data    []byte
}

// ParseMessage searches the byte slice for a message terminator and parses a message from the sequence of bytes
// it will return the number of bytes consumed
func ParseMessage(b []byte) (Message, int64, error) {
	ret := Message{}
	// Search for message terminator
	term := bytes.Index(b, MESSAGE_TERMINATOR)
	if term == -1 {
		return ret, 0, fmt.Errorf("invalid byte sequence")
	}

	return ret, int64(term) + int64(len(MESSAGE_TERMINATOR)), nil
}

type MessageReader struct {
	io.ReaderFrom
	io.Reader

	queue []Message
}

func NewMessageReader() MessageReader {
	return MessageReader{}
}

func (r *MessageReader) ReadFrom(rdr io.Reader) (n int64, err error) {
	b, err := io.ReadAll(rdr)
	if err != nil {
		return 0, err
	}

	for i := int64(0); i < int64(len(b)); {
		msg, n, err := ParseMessage(b[i:])
		if err != nil {
			return n, err
		}

		i += n
		r.queue = append(r.queue, msg)
	}

	return 0, nil
}
