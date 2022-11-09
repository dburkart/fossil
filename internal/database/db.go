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
	Version int
	Path    string

	// Private fields
	segments        []Segment
	current         int
	sharedLock      sync.Mutex
	exclusiveLock   sync.Mutex
	criticalSection bool
}

func (d *Database) appendInternal(data Datum) {
	if success, _ := d.segments[d.current].Append(data); !success {
		d.current += 1
		d.segments = append(d.segments, Segment{})
		d.segments[d.current].Append(data)
	}
}

func (d *Database) Append(data []byte) {
	e := Datum{Timestamp: time.Now(), Data: data}

	d.sharedLock.Lock()
	defer d.sharedLock.Unlock()

	log := WriteAheadLog{filepath.Join(d.Path, "wal.log")}
	log.AddEvent(&e)

	d.exclusiveLock.Lock()
	defer d.exclusiveLock.Unlock()
	// Using this variable seems race-y, but I'm not sure how to check the
	// state of a mutex in go
	d.criticalSection = true
	d.appendInternal(e)
	d.criticalSection = false
}

func NewDatabase(location string) *Database {
	db := Database{
		Version:  1,
		Path:     location,
		segments: []Segment{Segment{}},
		current:  0,
	}
	log := WriteAheadLog{filepath.Join(db.Path, "wal.log")}
	log.ApplyToDB(&db)
	return &db
}
