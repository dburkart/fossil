/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"encoding/base64"
	"fmt"
	"strings"
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
	Time   time.Time `json:"time"`
	Topic  string    `json:"topic"`
	Schema string    `json:"schema"`
	Data   []byte    `json:"data"`
}

func (e *Entry) ToString() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", e.Time.Format(time.RFC3339Nano), e.Topic, base64.StdEncoding.EncodeToString(e.Data), e.Schema)
}

func ParseEntry(s string) (Entry, error) {
	ent := Entry{}
	parts := strings.Split(s, "\t")
	if len(parts) < 3 {
		return ent, fmt.Errorf("malformed entry, expected 3 parts gpt %d", len(parts))
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return ent, err
	}
	ent.Time = t
	ent.Topic = parts[1]
	ent.Data, err = base64.StdEncoding.DecodeString(parts[2])
	ent.Schema = parts[3]
	return ent, nil
}

type Entries []Entry

// A Filter that takes a list of Datum and returns a filtered list of Datum.
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
