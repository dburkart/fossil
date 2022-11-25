package proto_test

import (
	"net"
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
)

func TestParseMessage(t *testing.T) {
	buf := proto.MESSAGE_TERMINATOR

	_, n, err := proto.ParseMessage(buf)
	if err != nil {
		t.Error(err)
	}
	if n != 2 {
		t.Errorf("should have read 2 bytes, read %d", n)
	}

	buf = append([]byte("INFO all"), proto.MESSAGE_TERMINATOR...)
	_, n, err = proto.ParseMessage(buf)
	if err != nil {
		t.Error(err)
	}
	if n != 10 {
		t.Errorf("should have read 10 bytes, read %d", n)
	}
}

func TestMessageReader(t *testing.T) {
	client, server := net.Pipe()

	// instance we are testing
	rdr := proto.NewMessageReader()

	// send some test data
	client.Write(append([]byte("INFO all"), proto.MESSAGE_TERMINATOR...))

	// test the data in the pipe
	conn := server.(*net.TCPConn)

	n, err := conn.ReadFrom(rdr)
	if err != nil {
		t.Error(err)
	}
	if n == 0 {
		t.Error("should read more than 0")
	}

}
