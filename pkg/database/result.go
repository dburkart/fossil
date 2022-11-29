/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import "time"

// Result wraps a slice of Items, and in the future will have other
// metadata stored within it as well.
type Result struct {
	Entries []Entry
}

// An Entry is a hydrated Datum, where the time and topic have been
// expanded.
type Entry struct {
	Time  time.Time
	Topic string
	Data  []byte
}
