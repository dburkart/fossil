/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"errors"
	"time"
)

const SegmentSize int = 10000
const IndexSize int = 100

type tsLookup struct {
	timestamp time.Time
	index     int
}

type Segment struct {
	HeadTime    time.Time
	Index       [IndexSize]tsLookup
	indexCursor int
	Size        int
	Series      [SegmentSize]Datum
}

func (s *Segment) Append(d Datum) (bool, error) {
	if s.Size >= SegmentSize {
		return false, errors.New("cannot add additional elements, segment at maximum size")
	}

	if s.Size == 0 {
		s.HeadTime = d.Timestamp
	}

	s.Series[s.Size] = d

	if s.indexCursor < IndexSize && (s.Size%(SegmentSize/IndexSize) == 0) {
		s.Index[s.indexCursor] = tsLookup{d.Timestamp, s.Size}
		s.indexCursor += 1
	}

	s.Size += 1

	return true, nil
}

func (s *Segment) Range(begin time.Time, end time.Time) []Datum {
	var startIndex, endIndex int
	// First, find where we must start in our segment
	if s.HeadTime.After(begin) {
		startIndex = 0
	} else {
		// TODO: This should be a binary search
		for i := 0; i < s.indexCursor; i++ {
			if s.Index[i].timestamp.After(begin) {
				startIndex = s.Index[i].index
				for j := s.Index[i-1].index; j < startIndex; j++ {
					if s.Series[j].Timestamp.After(begin) {
						startIndex = j
						break
					}
				}
				break
			}
		}
	}

	if s.Series[s.Size-1].Timestamp.Before(end) {
		endIndex = s.Size
	} else {
		// TODO: This should be a binary search
		for i := s.indexCursor - 1; i >= 0; i-- {
			if s.Index[i].timestamp.Before(end) {
				endIndex = s.Index[i].index
				for j := s.Index[i+1].index; j > endIndex; j-- {
					if s.Series[j].Timestamp.Before(end) {
						endIndex = j
						break
					}
				}
				break
			}
		}
	}

	return s.Series[startIndex:endIndex]
}
