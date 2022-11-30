/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import "testing"

func TestParseConnectionString(t *testing.T) {
	tt := []struct {
		test     string
		connStr  string
		addr     string
		local    bool
		database string
	}{
		{
			"Test empty conn string",
			"",
			"local",
			true,
			"default",
		},
		{
			"Test local no db",
			"fossil://local",
			"local",
			true,
			"default",
		},
		{
			"Test local db",
			"fossil://local/db1",
			"local",
			true,
			"db1",
		},
		{
			"Test host no db end slash",
			"fossil://localhost:8000/",
			"localhost:8000",
			false,
			"default",
		},
		{
			"Test no proto local no db",
			"local",
			"local",
			true,
			"default",
		},
		{
			"Test no proto local path db",
			"local/data/db.log",
			"local",
			true,
			"data/db.log",
		},
		{
			"Test no proto local no db",
			"localhost/",
			"localhost",
			false,
			"default",
		},
	}

	shouldPanic(t, func(t *testing.T) {
		ParseConnectionString("fosssil:///zx")
	})
	shouldPanic(t, func(t *testing.T) {
		ParseConnectionString("tcp:///zx")
	})

	for _, tc := range tt {
		t.Run(tc.test, func(t *testing.T) {
			connStr := ParseConnectionString(tc.connStr)
			recover()
			if connStr.Address != tc.addr {
				t.Errorf("Address mismatch: %s != %s", connStr.Address, tc.addr)
			}
			if connStr.Local != tc.local {
				t.Error("local mismatch")
			}
			if connStr.Database != tc.database {
				t.Errorf("database mismatch: %s != %s", connStr.Database, tc.database)
			}
		})
	}
}

func shouldPanic(t *testing.T, f func(t *testing.T)) {
	t.Helper()
	defer func() { _ = recover() }()
	f(t)
	t.Errorf("should have panicked")
}
