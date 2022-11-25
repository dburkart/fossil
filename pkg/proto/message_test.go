/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */
package proto_test

import (
	"testing"

	"github.com/dburkart/fossil/pkg/proto"
)

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
			_, err := proto.ParseMessage(tc.buf)
			if err != nil && !tc.err {
				t.Error(err)
			}
		})
	}
}
