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
			"./",
		},
		{
			"Test local no db",
			"file:///local",
			"local",
			true,
			"/local",
		},
		{
			"Test local db no scheme",
			"./local/db1",
			"local",
			true,
			"./local/db1",
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
			"local",
		},
		{
			"Test no proto local path db",
			"./data/default",
			"local",
			true,
			"./data/default",
		},
	}

	_, err := ParseConnectionString("fosssil:///zx")
	if err == nil {
		t.Error("fosssil:///zx should have caused an error")
	}

	_, err = ParseConnectionString("tcp:///zx")
	if err == nil {
		t.Error("tcp:///zx should have caused an error")
	}

	for _, tc := range tt {
		t.Run(tc.test, func(t *testing.T) {
			connStr, err := ParseConnectionString(tc.connStr)
			if err != nil {
				t.Error(err)
			}
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
