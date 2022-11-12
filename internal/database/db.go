/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Database struct {
	Version  int
	Path     string
	Segments []Segment
	Current  int

	// Private fields
	sharedLock      sync.Mutex
	exclusiveLock   sync.Mutex
	criticalSection bool
	appendCount     int
}

func (d *Database) appendInternal(data Datum) {
	if success, _ := d.Segments[d.Current].Append(data); !success {
		log.Fatal("We should never not have enough segments, since our write-ahead log creates them")
	}
	d.appendCount += 1
}

func (d *Database) splatToDisk() {
	var encoded bytes.Buffer

	// Stop all writes
	d.sharedLock.Lock()
	defer d.sharedLock.Unlock()

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(d)
	if err != nil {
		log.Fatal("encode:", err)
	}

	backupDBPath := filepath.Join(d.Path, "database.bak")
	file, err := os.OpenFile(backupDBPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.Write(encoded.Bytes())
	if err != nil {
		return
	}
	file.Close()

	// Enter the critical section, since we'll be zeroing out the WriteAheadLog
	d.exclusiveLock.Lock()
	defer d.exclusiveLock.Unlock()
	d.criticalSection = true
	// First, overwrite the database
	err = os.Rename(backupDBPath, filepath.Join(d.Path, "database"))
	if err != nil {
		log.Fatal(err)
	}
	// Next, zero out the WriteAheadLog
	err = os.Remove(filepath.Join(d.Path, "wal.log"))
	if err != nil {
		log.Fatal(err)
	}
	d.appendCount = 0
	d.criticalSection = false
}

func (d *Database) Append(data []byte) {
	// Pull our timestamp at the top
	appendTime := time.Now()

	// Calculate the delta
	delta := appendTime.Sub(d.Segments[d.Current].HeadTime)

	if d.appendCount > SegmentSize {
		d.splatToDisk()
	}

	d.sharedLock.Lock()
	defer d.sharedLock.Unlock()

	log := WriteAheadLog{filepath.Join(d.Path, "wal.log")}
	// Add a new segment to the log if needed
	if d.Segments[d.Current].Size >= SegmentSize {
		log.AddSegment(appendTime)
		delta = 0
	}
	e := Datum{Data: data, Delta: delta}
	log.AddEvent(&e)

	d.exclusiveLock.Lock()
	defer d.exclusiveLock.Unlock()
	// Using this variable seems race-y, but I'm not sure how to check the
	// state of a mutex in go
	d.criticalSection = true
	// Create a new segment if needed
	if d.Segments[d.Current].Size >= SegmentSize {
		d.Segments = append(d.Segments, Segment{HeadTime: appendTime})
		d.Current += 1
	}
	d.appendInternal(e)
	d.criticalSection = false
}

func NewDatabase(location string) *Database {
	var db Database
	if _, err := os.Stat(filepath.Join(location, "database")); err == nil {
		contents, err := ioutil.ReadFile(filepath.Join(location, "database"))
		if err != nil {
			log.Fatal(err)
		}

		dec := gob.NewDecoder(bytes.NewBuffer(contents))
		err = dec.Decode(&db)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		db = Database{
			Version:  1,
			Path:     location,
			Segments: []Segment{Segment{}},
			Current:  0,
		}
	}
	wal := WriteAheadLog{filepath.Join(db.Path, "wal.log")}
	wal.ApplyToDB(&db)
	if db.appendCount > SegmentSize {
		db.splatToDisk()
	}
	return &db
}
