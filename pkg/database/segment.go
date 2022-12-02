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

func absoluteDuration(x time.Duration) time.Duration {
	if x < 0 {
		return -x
	}
	return x
}

type Segment struct {
	HeadTime time.Time
	Series   [SegmentSize]Datum
	Size     int
}

func (s *Segment) Append(d *Datum) (bool, error) {
	if s.Size >= SegmentSize {
		return false, errors.New("cannot add additional elements, segment at maximum size")
	}

	if d.Delta == -1 {
		d.Delta = time.Now().Sub(s.HeadTime)
	}

	s.Series[s.Size] = *d
	s.Size += 1

	return true, nil
}

func (s *Segment) binarySearchApproximate(desired time.Duration, begin int, end int) (index int, proximity time.Duration) {
	var subIndex int
	var subProximity time.Duration

	if begin == end {
		return begin, s.Series[begin].Delta - desired
	}

	if end == begin+1 {
		if absoluteDuration(s.Series[begin].Delta-desired) < absoluteDuration(s.Series[end].Delta-desired) {
			return begin, s.Series[begin].Delta - desired
		} else {
			return end, s.Series[end].Delta - desired
		}
	}

	index = (begin + end) / 2
	proximity = s.Series[index].Delta - desired

	if proximity < 0 {
		subIndex, subProximity = s.binarySearchApproximate(desired, index, end)
	} else {
		subIndex, subProximity = s.binarySearchApproximate(desired, begin, index)
	}

	if absoluteDuration(proximity) > absoluteDuration(subProximity) {
		index = subIndex
		proximity = subProximity
	}

	return
}

func (s *Segment) FindApproximateDatum(desired time.Time) (int, Datum) {
	if desired.Before(s.HeadTime) {
		return 0, s.Series[0]
	}
	desiredDuration := desired.Sub(s.HeadTime)
	index, _ := s.binarySearchApproximate(desiredDuration, 0, s.Size-1)
	return index, s.Series[index]
}
