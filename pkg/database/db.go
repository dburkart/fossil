/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Database struct {
	Version  int
	Name     string
	Path     string
	Segments []Segment
	Current  int

	TopicLookup []string
	TopicCount  int

	// Private fields

	// Our topic map is marked private since it is not thread safe
	topics      map[string]int
	writeLock   sync.Mutex
	topicLock   sync.RWMutex
	appendCount int
	log         zerolog.Logger
}

func (d *Database) appendInternal(data *Datum) {
	if success, _ := d.Segments[d.Current].Append(data); !success {
		d.log.Fatal().Msg("We should never not have enough segments, since our write-ahead log creates them")
	}
	d.appendCount += 1
}

func normalizeTopicName(topicName string) string {
	if topicName == "" {
		topicName = "/"
	}

	if topicName[0] != '/' {
		topicName = "/" + topicName
	}

	return topicName
}

func (d *Database) addTopicInternal(topicName string) int {
	topicName = normalizeTopicName(topicName)
	index := d.TopicCount
	d.TopicLookup = append(d.TopicLookup, topicName)
	d.TopicCount += 1
	d.topicLock.Lock()
	defer d.topicLock.Unlock()
	d.topics[topicName] = index
	return index
}

func (d *Database) splatToDisk() {
	var encoded bytes.Buffer

	// Stop all writes
	d.writeLock.Lock()
	defer d.writeLock.Unlock()

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(d)
	if err != nil {
		d.log.Fatal().Err(err).Msg("error gob encoding database")
	}

	backupDBPath := filepath.Join(d.Path, "database.bak")
	file, err := os.OpenFile(backupDBPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		d.log.Fatal().Err(err).Msg("unable to openfile")
	}
	defer file.Close()

	_, err = file.Write(encoded.Bytes())
	if err != nil {
		return
	}
	file.Close()

	// First, overwrite the database
	err = os.Rename(backupDBPath, filepath.Join(d.Path, "database"))
	if err != nil {
		d.log.Fatal().Err(err).Msg("error renaming database file")
	}
	// Next, zero out the WriteAheadLog
	err = os.Remove(filepath.Join(d.Path, "wal.log"))
	if err != nil {
		d.log.Fatal().Err(err).Msg("error removing wal.log")
	}
	d.appendCount = 0
}

//-- Public Interfaces

func (d *Database) AddTopic(topic string) int {
	topic = normalizeTopicName(topic)

	d.topicLock.RLock()
	if index, exists := d.topics[topic]; exists {
		d.topicLock.RUnlock()
		return index
	}
	d.topicLock.RUnlock()

	// The topic doesn't exist, so add it
	d.writeLock.Lock()
	defer d.writeLock.Unlock()

	index := d.addTopicInternal(topic)
	wal := WriteAheadLog{filepath.Join(d.Path, "wal.log")}
	wal.AddTopic(topic)

	return index
}

// Append to the end of the database
func (d *Database) Append(data []byte, topic string) {
	// Pull our timestamp at the top
	appendTime := time.Now()

	topicID := d.AddTopic(topic)

	// Explicitly copy the data before taking the lock to minimize resource
	// contention
	e := Datum{Data: make([]byte, len(data)), TopicID: topicID}
	copy(e.Data, data)

	if d.appendCount > SegmentSize {
		d.splatToDisk()
	}

	d.writeLock.Lock()
	defer d.writeLock.Unlock()

	wal := WriteAheadLog{filepath.Join(d.Path, "wal.log")}

	// Add a new segment to the log if needed
	if d.Segments[d.Current].Size >= SegmentSize {
		wal.AddSegment(appendTime)
		d.Segments = append(d.Segments, Segment{HeadTime: appendTime})
		d.Current += 1
	}
	if len(d.Segments) == 0 {
		wal.AddSegment(appendTime)
		d.Segments = append(d.Segments, Segment{HeadTime: appendTime})
	}

	// Calculate the delta
	delta := appendTime.Sub(d.Segments[d.Current].HeadTime)
	e.Delta = delta
	wal.AddEvent(&e)
	d.appendInternal(&e)
}

func (d *Database) entriesFromData(s *Segment, data []Datum) []Entry {
	entries := make([]Entry, len(data), cap(data))

	for index, val := range data {
		entries[index] = Entry{
			Time:  s.HeadTime.Add(val.Delta),
			Topic: d.TopicLookup[val.TopicID],
			Data:  val.Data,
		}
	}

	return entries
}

// Retrieve a list of datum from the database matching some query
// TODO: Eventually, this should return a proper result set
func (d *Database) Retrieve(q Query) []Entry {
	results := make([]Entry, 0)
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

		// If start has not been found, we still need to search the last segment
		// of the database
		if !startFound {
			startIndex = d.Current
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
		// End of the range should be inclusive
		endSubIndex += 1

		// Our binary search is kind of crude, in that it "fuzzy" matches the start
		// and end of our range. So we have to do a quick bounds check on both sides
		// to make sure that q.Range.Start <= startSubIndex <= q.Range.End
		switch q.RangeSemantics {
		case "since":
			// Ensure start is correct
			startDatum := d.Segments[startIndex].Series[startSubIndex]
			startTime := d.Segments[startIndex].HeadTime.Add(startDatum.Delta)
			if startTime.Before(q.Range.Start) {
				startSubIndex += 1
			}
		case "before":
			// Ensure end is correct
			endDatum := d.Segments[endIndex].Series[endSubIndex]
			endTime := d.Segments[startIndex].HeadTime.Add(endDatum.Delta)
			if endTime.After(q.Range.End) {
				endSubIndex -= 1
			}
		}
	}

	// Handle the case where all of our datum is in a single segment
	if startIndex == endIndex {
		segment := d.Segments[startIndex]
		data := segment.Series[startSubIndex:endSubIndex]
		return d.entriesFromData(&segment, data)
	}

	// Since our start and end are different segments, build a result set
	for i := startIndex; i <= endIndex; i++ {
		segment := d.Segments[i]
		if i == startIndex {
			data := segment.Series[startSubIndex:]
			results = append(results, d.entriesFromData(&segment, data)...)
		} else if i == endIndex {
			data := segment.Series[:endSubIndex]
			results = append(results, d.entriesFromData(&segment, data)...)
		} else {
			data := segment.Series[:]
			results = append(results, d.entriesFromData(&segment, data)...)
		}
	}

	return results
}

// NewDatabase creates a new database object in memory and creates the
// directory and files on disk for storing the data
// location is the base directory for creating the database
func NewDatabase(log zerolog.Logger, name string, location string) (*Database, error) {
	var db Database

	directory := filepath.Join(location, name)

	// If the path does not exist, create a new directory
	fileinfo, err := os.Stat(directory)
	if os.IsNotExist(err) {
		err := os.Mkdir(directory, 0700)
		if err != nil {
			return nil, err
		}
	} else if !fileinfo.IsDir() {
		return nil, fmt.Errorf("supplied path is not a directory")
	}

	if _, err := os.Stat(filepath.Join(directory, "database")); err == nil {
		contents, err := os.ReadFile(filepath.Join(directory, "database"))
		if err != nil {
			return nil, err
		}

		dec := gob.NewDecoder(bytes.NewBuffer(contents))
		err = dec.Decode(&db)
		if err != nil {
			return nil, err
		}
	} else if _, err := os.Stat(filepath.Join(directory, "wal.log")); err == nil {
		db = Database{
			Version:    1,
			Path:       directory,
			Segments:   []Segment{},
			Current:    0,
			topics:     make(map[string]int),
			TopicCount: 0,
		}
		wal := WriteAheadLog{filepath.Join(db.Path, "wal.log")}
		wal.ApplyToDB(&db)
	} else {
		db = Database{
			Version:    1,
			Path:       directory,
			Segments:   []Segment{},
			Current:    0,
			topics:     make(map[string]int),
			TopicCount: 0,
		}
		db.AddTopic("/")
		// TODO: Generalize this
		sTime := time.Now()
		wal := WriteAheadLog{filepath.Join(db.Path, "wal.log")}
		wal.AddSegment(sTime)
		db.Segments = append(db.Segments, Segment{HeadTime: sTime})
	}
	// We set the name here so that it's always correct, since the name can
	// change after we first splat to disk.
	db.Name = name
	if db.appendCount > SegmentSize {
		db.splatToDisk()
	}
	// Set up our convenience topic map
	for k := range db.TopicLookup {
		db.topics[db.TopicLookup[k]] = k
	}
	return &db, nil
}
