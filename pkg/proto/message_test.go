/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/dburkart/fossil/pkg/database"
)

var result Message

func TestMessageMarshaling(t *testing.T) {
	m := NewMessageWithType(CommandAppend, AppendRequest{Topic: "", Data: []byte("y2k")})
	b, err := m.Marshal()
	if err != nil {
		t.Fail()
	}
	if len(b) != 4+8+4+3 {
		t.Fail()
	}

	err = m.Unmarshal(bytes.NewBuffer(b))
	if err != nil {
		t.Fail()
	}

	// Check fields
	if m.Command != CommandAppend {
		t.Fail()
	}
	if !bytes.Equal(m.Data, []byte("\u0000\u0000\u0000\u0000y2k")) {
		t.Fail()
	}
}

func BenchmarkReadMessageFull(b *testing.B) {
	buf := new(bytes.Buffer)
	rw := NewResponseWriter(buf)
	rw.WriteMessage(MessageErrorCommandNotFound)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ret, _ := ReadMessageFull(buf)
		result = ret
	}
}

func TestUseRequest(t *testing.T) {
	req := UseRequest{}

	b, _ := req.Marshal()
	if !bytes.Equal(b, []byte{}) {
		t.Fail()
	}
	req = UseRequest{DbName: "Tester"}

	b, _ = req.Marshal()
	if !bytes.Equal(b, []byte("Tester")) {
		t.Fail()
	}

	err := req.Unmarshal(b)
	if err != nil {
		t.Fail()
	}

	// Check fields
	if req.DbName != "Tester" {
		t.Fail()
	}
}

func TestOkResponse(t *testing.T) {
	req := OkResponse{Code: 200, Message: "test"}

	b, _ := req.Marshal()
	err := req.Unmarshal(b)
	if err != nil {
		t.Fail()
	}

	// Check fields
	if req.Code != 200 {
		t.Fail()
	}
	if req.Message != "test" {
		t.Fail()
	}
}

func TestErrResponse(t *testing.T) {
	req := ErrResponse{Code: 500, Err: errors.New("test")}

	b, _ := req.Marshal()
	err := req.Unmarshal(b)
	if err != nil {
		t.Fail()
	}

	// Check fields
	if req.Code != 500 {
		t.Fail()
	}
	if req.Err.Error() != "test" {
		t.Fail()
	}
}

func TestAppendRequest(t *testing.T) {
	t.Run("empty topic", func(t *testing.T) {
		req := AppendRequest{Topic: "", Data: []byte("woohoo")}
		b, _ := req.Marshal()
		err := req.Unmarshal(b)
		if err != nil {
			t.Fail()
		}

		// Check fields
		if req.Topic != "/" {
			t.Fail()
		}
		if !bytes.Equal(req.Data, []byte("woohoo")) {
			t.Fail()
		}

	})

	t.Run("topic and data", func(t *testing.T) {
		req := AppendRequest{Topic: "/path/of/the/gods", Data: []byte("woohoo")}

		b, _ := req.Marshal()
		err := req.Unmarshal(b)
		if err != nil {
			t.Fail()
		}

		// Check fields
		if req.Topic != "/path/of/the/gods" {
			t.Fail()
		}
		if !bytes.Equal(req.Data, []byte("woohoo")) {
			t.Fail()
		}
	})
}

func TestQueryRequest(t *testing.T) {
	req := QueryRequest{Query: "all"}

	b, _ := req.Marshal()
	err := req.Unmarshal(b)
	if err != nil {
		t.Fail()
	}

	// Check fields
	if req.Query != "all" {
		t.Fail()
	}
}

func TestQueryResponse(t *testing.T) {
	req := QueryResponse{Results: database.Entries{}}

	b, _ := req.Marshal()
	err := req.Unmarshal(b)
	if err != nil {
		t.Fail()
	}

	// Check fields
	if len(req.Results) != 0 {
		t.Fail()
	}

	testTime := time.Date(2000, 1, 1, 1, 1, 1, 1, time.Local)
	req = QueryResponse{Results: database.Entries{
		{
			Time:  testTime,
			Topic: "/",
			Data:  []byte("y2k"),
		},
	}}

	b, _ = req.Marshal()
	err = req.Unmarshal(b)
	if err != nil {
		t.Fail()
	}

	// Check fields
	// 2 because there is a root segment
	if len(req.Results) != 2 {
		t.Fail()
	}
	if !req.Results[0].Time.Equal(testTime) {
		t.Fail()
	}
	if req.Results[0].Topic != "/" {
		t.Fail()
	}
	if !bytes.Equal(req.Results[0].Data, []byte("y2k")) {
		t.Fail()
	}
}

func TestStatsRequest(t *testing.T) {
	req := StatsRequest{Database: "default"}

	b, _ := req.Marshal()
	err := req.Unmarshal(b)
	if err != nil {
		t.Fail()
	}

	// Check fields
	if req.Database != "default" {
		t.Fail()
	}
}

func TestStatsResponse(t *testing.T) {
	req := StatsResponse{AllocHeap: 123, TotalMem: 123, Uptime: 10 * time.Hour, Segments: 10}

	b, _ := req.Marshal()
	err := req.Unmarshal(b)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	// Check fields
	if req.AllocHeap != uint64(123) {
		t.Fail()
	}
	if req.TotalMem != uint64(123) {
		t.Fail()
	}
	if req.Uptime != 10*time.Hour {
		t.Fail()
	}
	if req.Segments != 10 {
		t.Fail()
	}
}
