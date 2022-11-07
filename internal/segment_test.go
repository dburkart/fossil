/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package internal

import (
	"testing"
	"time"
)

func TestAddingToSegment(t *testing.T) {
	segment := Segment{}

	location, _ := time.LoadLocation("America/Los_Angeles")
	event := Event{
		Timestamp: time.Date(2022, 11, 7, 13, 0, 0, 0, location),
		Data:      "{\"foo\": 12}",
	}

	for i := 0; i < int(SegmentSize); i++ {
		if result, err := segment.AddEvent(event); !result {
			t.Errorf("got error when adding event: %s", err.Error())
		}
		event.Timestamp = event.Timestamp.Add(60000000000)
	}

	// Ensure that there are 10000 events in our segment
	if segment.Size != SegmentSize {
		t.Errorf("expected 10,000 events, but only found %d", segment.Size)
	}

	// Ensure that there is 1,000 indices
	if segment.Index[IndexSize-1].index == 0 {
		t.Errorf("expected to find 1,000 indices")
	}
}
