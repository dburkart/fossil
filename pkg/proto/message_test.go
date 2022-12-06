/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import (
	"bytes"
	"testing"
)

var result Message

func TestParseMessage(t *testing.T) {
	tt := []struct {
		test string
		buf  []byte
		err  bool
	}{
		{
			"Test empty message",
			[]byte("\r\n"),
			true,
		},
		{
			"Test simple message",
			[]byte("INFO all\n\n\n"),
			false,
		},
		{
			"Test simple message",
			[]byte("INFO all"),
			false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.test, func(t *testing.T) {
			_, err := ParseMessage(tc.buf)
			if err != nil && !tc.err {
				t.Error(err)
			}
		})
	}
}

func BenchmarkReadMessage(b *testing.B) {
	buf := new(bytes.Buffer)
	rw := NewResponseWriter(buf)
	rw.WriteMessage(MessageErrorCommandNotFound)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ret, _ := ReadMessage(buf)
		result = ret
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
