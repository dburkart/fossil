/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */
package proto_test

import (
	"bytes"
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
)

func TestParseMessage(t *testing.T) {
	tt := []struct {
		test string
		buf  []byte
		n    int64
	}{
		{
			"Test empty message",
			proto.MESSAGE_TERMINATOR,
			2,
		},
		{
			"Test simple message",
			append([]byte("INFO all"), proto.MESSAGE_TERMINATOR...),
			10,
		},
	}

	for _, tc := range tt {
		t.Run(tc.test, func(t *testing.T) {
			_, n, err := proto.ParseMessage(tc.buf)
			if err != nil {
				t.Error(err)
			}
			if n != tc.n {
				t.Errorf("should have read %d bytes, read %d", tc.n, n)
			}
		})
	}
}

func TestMessageReader(t *testing.T) {
	tt := []struct {
		test     string
		buf      []byte
		n        int64
		msgCount int64
	}{
		{
			"Test empty message",
			proto.MESSAGE_TERMINATOR,
			2,
			1,
		},
		{
			"Test simple message",
			append([]byte("INFO all"), proto.MESSAGE_TERMINATOR...),
			10,
			1,
		},
	}

	for _, tc := range tt {
		t.Run(tc.test, func(t *testing.T) {
			bufReader := bytes.NewBuffer(tc.buf)
			reader := proto.NewMessageReader()
			n, err := reader.ReadFrom(bufReader)

			if err != nil {
				t.Error(err)
			}
			if n == tc.n {
				t.Error("should read more than 0")
			}
			msgCount := len(reader.PopMessages())
			if msgCount != int(tc.msgCount) {
				t.Errorf("Expected %d message, got %d", msgCount, tc.msgCount)
			}
		})
	}
}
