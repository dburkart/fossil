/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package internal

import (
	"sync"
	"time"
)

type Database struct {
	Segments        []Segment
	Current         int
	SharedLock      sync.Mutex
	ExclusiveLock   sync.Mutex
	criticalSection bool
}

func (d *Database) Add(data EventData) {
	e := Event{time.Now(), data}

	d.SharedLock.Lock()
	defer d.SharedLock.Unlock()

	// TODO: Write-Ahead logging here

	d.ExclusiveLock.Lock()
	defer d.ExclusiveLock.Unlock()
	d.criticalSection = true

	if success, _ := d.Segments[d.Current].Add(e); !success {
		d.Current += 1
		d.Segments[d.Current] = Segment{}
		d.Segments[d.Current].Add(e)
	}

	d.criticalSection = false
}

func NewDatabase() *Database {
	return &Database{
		Segments: []Segment{Segment{}},
		Current:  0,
	}
}
