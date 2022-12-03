/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package server

import (
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
)

func stub2(rw proto.ResponseWriter, msg *proto.Request) {

}

func BenchmarkMapCommandParse(b *testing.B) {
	mux := NewMapMux()

	mux.Handle("A", stub2)
	mux.Handle("B", stub2)
	mux.Handle("C", stub2)

	tests := []*proto.Request{proto.NewRequest(proto.Message{
		Command: "A",
	}, nil), proto.NewRequest(proto.Message{
		Command: "B",
	}, nil), proto.NewRequest(proto.Message{
		Command: "C",
	}, nil),
	}

	c := &conn{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeMessage(c, tests[i%len(tests)])
	}
}
