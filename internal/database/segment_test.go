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

	event := Event{
		Timestamp: startTime(),
		Data:      "{\"foo\": 12}",
	}

	for i := 0; i < int(SegmentSize); i++ {
		segment.Add(event)
		event.Timestamp = event.Timestamp.Add(60000000000)
	}

	return segment
}

func TestAddingToSegment(t *testing.T) {
	segment := createFullSegment()

	// Ensure that there are 10000 events in our segment
	if segment.Size != SegmentSize {
		t.Errorf("expected 10,000 events, but only found %d", segment.Size)
	}

	// Ensure that there is 1,000 indices
	if segment.Index[IndexSize-1].index == 0 {
		t.Errorf("expected to find 1,000 indices")
	}
}

func TestRangeFunctionality(t *testing.T) {
	segment := createFullSegment()

	events := segment.Range(startTime(), startTime().Add(time.Hour))

	if len(events) != 58 {
		t.Errorf("expected 58 events, got %d", len(events))
	}
}
