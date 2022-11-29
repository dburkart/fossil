/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"fmt"
	"time"
)

// Result wraps a slice of Items.
// TODO: Track query statistics and the like in here
type Result struct {
	Data Entries
}

// An Entry is a hydrated Datum, where the time and topic have been
// expanded.
type Entry struct {
	Time  time.Time
	Topic string
	Data  []byte
}

func (e *Entry) ToString() string {
	return fmt.Sprintf("%s\t%s\t%s", e.Time, e.Topic, string(e.Data))
}

type Entries []Entry

// A Filter that takes a list of Datum and returns a filtered lsit of Datum.
type Filter func(Entries) Entries

type Filters []Filter

func (f *Filters) Execute() Result {
	var entries Entries = nil

	for i := len(*f) - 1; i >= 0; i-- {
		entries = (*f)[i](entries)
	}

	return Result{
		Data: entries,
	}
}
