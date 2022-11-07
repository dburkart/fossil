/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package internal

import (
	"errors"
	"time"
)

const SegmentSize int64 = 10000
const IndexSize int64 = 100

type tsLookup struct {
	timestamp time.Time
	index     int64
}

type Segment struct {
	HeadTime    time.Time
	Index       [IndexSize]tsLookup
	indexCursor int64
	Size        int64
	Series      [SegmentSize]Event
}

func (s *Segment) AddEvent(e Event) (bool, error) {
	if s.Size >= SegmentSize {
		return false, errors.New("cannot add additional elements, segment at maximum size")
	}

	if s.Size == 0 {
		s.HeadTime = e.Timestamp
	}

	s.Series[s.Size] = e

	if s.indexCursor < IndexSize && (s.Size%(SegmentSize/IndexSize) == 0) {
		s.Index[s.indexCursor] = tsLookup{e.Timestamp, s.Size}
		s.indexCursor += 1
	}

	s.Size += 1

	return true, nil
}
