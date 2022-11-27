package server_test

import (
	"io"
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
	"github.com/dburkart/fossil/pkg/server"
)

func stub2(rw io.Writer, msg proto.Message) {

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
