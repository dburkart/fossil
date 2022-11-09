/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"path/filepath"
	"sync"
	"time"
)

type Database struct {
	Path            string
	Segments        []Segment
	Current         int
	SharedLock      sync.Mutex
	ExclusiveLock   sync.Mutex
	criticalSection bool
}

func (d *Database) Append(data OpaqueData) {
	e := Datum{time.Now(), data}

	d.SharedLock.Lock()
	defer d.SharedLock.Unlock()

	log := WriteAheadLog{filepath.Join(d.Path, "wal.log")}
	log.AddEvent(&e)

	d.ExclusiveLock.Lock()
	defer d.ExclusiveLock.Unlock()
	// Using this variable seems race-y, but I'm not sure how to check the
	// state of a mutex in go
	d.criticalSection = true

	if success, _ := d.Segments[d.Current].Append(e); !success {
		d.Current += 1
		d.Segments[d.Current] = Segment{}
		d.Segments[d.Current].Append(e)
	}

	d.criticalSection = false
}

func NewDatabase(location string) *Database {
	return &Database{
		Path:     location,
		Segments: []Segment{Segment{}},
		Current:  0,
	}
}
