/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"testing"
	"time"
)

func startTime() time.Time {
	location, _ := time.LoadLocation("America/Los_Angeles")
	return time.Date(2022, 11, 7, 13, 0, 0, 0, location)
}

func createFullSegment() Segment {
	segment := Segment{}

	event := Datum{
		Data:  []byte("{\"foo\": 12}"),
		Delta: 0,
	}

	for i := 0; i < SegmentSize; i++ {
		segment.Append(&event)
		event.Delta += 60000000000
	}

	segment.HeadTime = startTime()

	return segment
}

func TestAddingToSegment(t *testing.T) {
	segment := createFullSegment()

	// Ensure that there are 10000 events in our segment
	if segment.Size != SegmentSize {
		t.Errorf("expected 10,000 events, but only found %d", segment.Size)
	}
}

func TestBinarySearch(t *testing.T) {
	segment := createFullSegment()

	index, _ := segment.FindApproximateDatum(startTime().Add(120000012345))

	if index != 2 {
		t.Errorf("expected value at index 2, but only got %d", index)
	}
}
