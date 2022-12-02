/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import "time"

type TimeRange struct {
	Start time.Time
	End   time.Time
}

// The Query object represents a single query on a database. It contains the
// 4 main variables of a query:
//   - Quantifier
//   - Topic(s)
//   - Time Range
//   - Data Predicate (TODO!)
type Query struct {
	Quantifier     string
	Topics         []string
	Range          *TimeRange // nil means entire history (no time range)
	RangeSemantics string     // none, before, since, between
}
