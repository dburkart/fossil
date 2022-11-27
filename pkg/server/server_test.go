package server_test

import (
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
)

func BenchmarkSliceCommandParse(b *testing.B) {
	mux := Mux{
		mapping: make([]HandleFunc, 0, 10),
	}
	mux.Handle("A", stub)
	mux.Handle("B", stub)
	mux.Handle("C", stub)

	tests := []proto.Message{{
		Command: "A",
	}, {
		Command: "B",
	}, {
		Command: "C",
	},
	}

	for i := 0; i < b.N; i++ {
		testMsg := tests[i%len(tests)]
		ret := mux.Serve(testMsg)
		if ret != testMsg.Command {
			b.Errorf("Incorrect response for %s got %s", testMsg.Command, ret)
		}
	}
}

type Mux struct {
	mapping []HandleFunc
}

type HandleFunc func(msg proto.Message) string

func (m *Mux) Handle(command string, f HandleFunc) {
	h := hash(command)
	if h >= len(m.mapping) {
		temp := m.mapping
		m.mapping = make([]HandleFunc, h+1, h+1)
		copy(m.mapping, temp)
	}
	m.mapping[hash(command)] = f
}

func (m *Mux) Serve(msg proto.Message) string {
	cmd := hash(msg.Command)
	if len(m.mapping) < cmd {
		return ""
	}

	return m.mapping[cmd](msg)
}

func stub(msg proto.Message) string {
	return msg.Command
}

func hash(s string) int {
	return int(s[0])
}
