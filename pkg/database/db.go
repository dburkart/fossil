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
	Version    int
	Path       string
	Segments   []Segment
	Current    int
	Topics     map[string]int
	TopicCount int

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

func (d *Database) addTopic(topicName string) int {
	if topicName == "" {
		topicName = "/"
	}

	if topicName[0] != '/' {
		topicName = "/" + topicName
	}

	if index, exists := d.Topics[topicName]; exists {
		return index
	}
	index := d.TopicCount
	d.TopicCount += 1
	d.Topics[topicName] = index
	return index
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

//-- Public Interfaces

// Append to the end of the database
func (d *Database) Append(data []byte, topic string) {
	// Pull our timestamp at the top
	appendTime := time.Now()

	topicID := d.addTopic(topic)

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
	e := Datum{Data: data, TopicID: topicID, Delta: delta}
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

// Retrieve a list of datum from the database matching some query
// TODO: Eventually, this should return a proper result set
func (d *Database) Retrieve(q Query) []Datum {
	results := make([]Datum, 0)
	// First, we deal with the time range
	startFound := false
	startIndex := 0

	endFound := false
	endIndex := 0

	// If the query range is nil, we can skip this
	if q.Range != nil {
		for index, segment := range d.Segments {
			if !startFound && segment.HeadTime.After(q.Range.Start) {
				if index > 0 {
					startIndex = index - 1
				}
				startFound = true
			}

			if !endFound && segment.HeadTime.After(q.Range.End) {
				if index > 0 {
					endIndex = index - 1
				} else {
					return results
				}
				endFound = true
			}
		}

		// If we haven't found a start to our range, then it's outside the
		// bounds of our database.
		if !startFound {
			return results
		}
	}

	// If endIndex is 0, that means there are no segments with head times after
	// the specified end time, so use the last segment
	if endIndex == 0 {
		endIndex = d.Current
	}

	startSubIndex := 0
	endSubIndex := d.Segments[endIndex].Size

	if q.Range != nil {
		startSubIndex, _ = d.Segments[startIndex].FindApproximateDatum(q.Range.Start)
		endSubIndex, _ = d.Segments[endIndex].FindApproximateDatum(q.Range.End)
	}

	// Handle the case where all of our datum is in a single segment
	if startIndex == endIndex {
		return d.Segments[startIndex].Series[startSubIndex:endSubIndex]
	}

	// Since our start and end are different segments, build a result set
	for i := startIndex; i <= endIndex; i++ {
		if i == startIndex {
			results = append(results, d.Segments[i].Series[startSubIndex:]...)
		} else if i == endIndex {
			results = append(results, d.Segments[i].Series[:endSubIndex]...)
		} else {
			results = append(results, d.Segments[i].Series[:]...)
		}
	}

	return results
}

func NewDatabase(location string) *Database {
	var db Database
	// If the path does not exist, create a new directory
	fileinfo, err := os.Stat(location)
	if os.IsNotExist(err) {
		err := os.Mkdir(location, 0700)
		if err != nil {
			log.Fatal(err)
		}
	} else if !fileinfo.IsDir() {
		log.Fatal("Supplied path is not a directory")
	}

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
			Version:    1,
			Path:       location,
			Segments:   []Segment{{}},
			Current:    0,
			Topics:     make(map[string]int),
			TopicCount: 0,
		}
		db.addTopic("/")
	}
	wal := WriteAheadLog{filepath.Join(db.Path, "wal.log")}
	wal.ApplyToDB(&db)
	if db.appendCount > SegmentSize {
		db.splatToDisk()
	}
	return &db
}
