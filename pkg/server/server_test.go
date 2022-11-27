package server_test

import (
	"io"
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/server"
)

func BenchmarkSliceCommandParse(b *testing.B) {
	mux := server.NewSliceMux()
	mux.Handle("A", stub2)
	mux.Handle("B", stub2)
	mux.Handle("C", stub2)

	tests := []proto.Message{{
		Command: "A",
	}, {
		Command: "B",
	}, {
		Command: "C",
	},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMsg := tests[i%len(tests)]
		mux.ServeMessage(io.Discard, testMsg)
	}
}

func stub2(rw io.Writer, msg proto.Message) {

}

func BenchmarkSwitchCommandParse(b *testing.B) {
	mux := func(msg proto.Message) {
		switch msg.Command {
		case "A":
			stub2(io.Discard, msg)
		case "B":
			stub2(io.Discard, msg)
		case "C":
			stub2(io.Discard, msg)
		}
	}

	tests := []proto.Message{{
		Command: "A",
	}, {
		Command: "B",
	}, {
		Command: "C",
	},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMsg := tests[i%len(tests)]
		mux(testMsg)
	}
}

func BenchmarkMapCommandParse(b *testing.B) {
	mux := server.NewMapMux()

	mux.Handle("A", stub2)
	mux.Handle("B", stub2)
	mux.Handle("C", stub2)

	tests := []proto.Message{{
		Command: "A",
	}, {
		Command: "B",
	}, {
		Command: "C",
	},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMsg := tests[i%len(tests)]
		mux.ServeMessage(io.Discard, testMsg)
	}
}
