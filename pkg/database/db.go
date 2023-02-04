/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/dburkart/fossil/pkg/schema"
	"github.com/rs/zerolog"
)

// FossilDBVersion is the version of the database as recorded on disk.
// This is primarily used for migration.
const FossilDBVersion = 2

type Database struct {
	Version      uint32
	Segments     []Segment
	Current      uint32
	TopicLookup  []string
	SchemaLookup []schema.Object
	TopicCount   int
	STime        time.Time // Last serialize time
	Name         string    // <-- We do not save to disk, starting here
	Path         string

	// Private fields

	// Our topic map is marked private since it is not thread safe
	topics      map[string]int
	schemaCache sync.Map
	writeLock   sync.Mutex
	topicLock   sync.RWMutex
	appendCount int
	log         zerolog.Logger
}

func (db *Database) Stats() Stats {
	return Stats{
		Segments:      len(db.Segments),
		TopicCount:    db.TopicCount,
		SerializeTime: db.STime,
	}
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

	if len(topicName) > 1 && topicName[len(topicName)-1] == '/' {
		topicName = topicName[:len(topicName)-1]
	}

	return topicName
}

// parentSchema returns the first non-string schema in any parent of topic, or nil
func (d *Database) parentSchema(topicName string) schema.Object {
	if topicName == "/" {
		return nil
	}

	d.topicLock.RLock()
	idx, ok := d.topics[topicName]
	d.topicLock.RUnlock()

	if ok {
		schemaObj := d.SchemaLookup[idx]
		if t, isType := schemaObj.(schema.Type); isType && t.Name == "string" {
			return d.parentSchema(path.Dir(topicName))
		}
		return schemaObj
	}

	return d.parentSchema(path.Dir(topicName))
}

func (d *Database) loadSchema(s string) schema.Object {
	obj, ok := d.schemaCache.Load(s)
	if !ok {
		o, err := schema.Parse(s)
		if err != nil {
			obj, ok = d.schemaCache.Load("string")
			if !ok {
				obj, _ = schema.Parse("string")
			}
		} else {
			d.schemaCache.Store(s, o)
			return o
		}
	}
	return obj.(schema.Object)
}

func (d *Database) addTopicInternal(topicName string, s string) int {
	topicName = normalizeTopicName(topicName)
	index := d.TopicCount
	d.SchemaLookup = append(d.SchemaLookup, d.loadSchema(s))
	d.TopicLookup = append(d.TopicLookup, topicName)
	d.TopicCount += 1
	d.topicLock.Lock()
	defer d.topicLock.Unlock()
	d.topics[topicName] = index
	return index
}

// deserializeInternal de-serializes a database from disk.
// It expects the path field to be filled in on the database struct
func (db *Database) deserializeInternal() error {
	// First, read in our metadata
	file, err := os.Open(path.Join(db.Path, "metadata"))
	if err != nil {
		return err
	}
	defer file.Close()

	r := bufio.NewReader(file)
	err = binary.Read(r, binary.LittleEndian, &db.Version)
	if err != nil {
		return err
	}
	if db.Version > FossilDBVersion {
		return errors.New(fmt.Sprintf("cannot read database, on-disk version (%d) is greater than our version (%d)", db.Version, FossilDBVersion))
	}

	var segmentCount uint32
	err = binary.Read(r, binary.LittleEndian, &segmentCount)
	if err != nil {
		return err
	}

	err = binary.Read(r, binary.LittleEndian, &db.Current)
	if err != nil {
		return err
	}

	timeBytes, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	db.STime, err = time.Parse(time.RFC3339, string(timeBytes))
	if err != nil {
		return err
	}

	segmentsDirectory := path.Join(db.Path, "segments")
	for i := uint32(0); i < segmentCount; i++ {
		var segment Segment

		contents, err := os.ReadFile(filepath.Join(segmentsDirectory, fmt.Sprintf("%d", i)))
		if err != nil {
			return err
		}

		dec := gob.NewDecoder(bytes.NewBuffer(contents))
		err = dec.Decode(&segment)
		if err != nil {
			return err
		}

		db.Segments = append(db.Segments, segment)
	}

	file, err = os.Open(path.Join(db.Path, "topics"))
	if err != nil {
		return err
	}

	reader, err := zlib.NewReader(file)
	if err != nil {
		return err
	}

	var topicBuffer bytes.Buffer
	_, err = io.Copy(&topicBuffer, reader)
	if err != nil {
		return err
	}

	err = json.Unmarshal(topicBuffer.Bytes(), &db.TopicLookup)
	if err != nil {
		return err
	}

	file, err = os.Open(path.Join(db.Path, "schemas"))
	if err != nil {
		return err
	}
	reader.Close()

	reader, err = zlib.NewReader(file)
	if err != nil {
		return err
	}

	var schemaBuffer bytes.Buffer
	_, err = io.Copy(&schemaBuffer, reader)

	var schemas []string
	err = json.Unmarshal(schemaBuffer.Bytes(), &schemas)
	if err != nil {
		return err
	}

	for _, s := range schemas {
		db.SchemaLookup = append(db.SchemaLookup, db.loadSchema(s))
	}

	db.TopicCount = len(db.TopicLookup)
	return nil
}

func (db *Database) serializeInternal() error {
	// First, we write out our database metadata
	newSTime := time.Now()
	databaseMetadata := bytes.NewBuffer(binary.LittleEndian.AppendUint32([]byte{}, db.Version))
	_, err := databaseMetadata.Write(binary.LittleEndian.AppendUint32([]byte{}, uint32(len(db.Segments))))
	if err != nil {
		return err
	}
	_, err = databaseMetadata.Write(binary.LittleEndian.AppendUint32([]byte{}, db.Current))
	if err != nil {
		return err
	}
	_, err = databaseMetadata.Write([]byte(newSTime.Format(time.RFC3339)))
	if err != nil {
		return err
	}

	// Ensure that there is a segments directory
	segmentsDirectory := path.Join(db.Path, "segments")
	_, err = os.Stat(segmentsDirectory)
	if os.IsNotExist(err) {
		err = os.Mkdir(segmentsDirectory, 0755)
		if err != nil {
			return err
		}
	}

	// Now, write out any segments after our STime
	var first int
	for idx, s := range db.Segments {
		if s.HeadTime.After(db.STime) {
			first = idx
			if idx > 0 {
				first = idx - 1
			}
			break
		}
	}

	for i := uint32(first); i <= db.Current; i++ {
		var encoded bytes.Buffer

		enc := gob.NewEncoder(&encoded)
		err := enc.Encode(db.Segments[i])
		if err != nil {
			db.log.Fatal().Err(err).Msg("error encoding segment")
		}

		tmpPath := filepath.Join(segmentsDirectory, fmt.Sprintf("%d.tmp", i))
		file, err := os.OpenFile(tmpPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.Write(encoded.Bytes())
		if err != nil {
			return err
		}
		file.Close()
	}

	for i := uint32(first); i <= db.Current; i++ {
		err = os.Rename(path.Join(segmentsDirectory, fmt.Sprintf("%d.tmp", i)), path.Join(segmentsDirectory, fmt.Sprintf("%d", i)))
		if err != nil {
			return err
		}
	}

	// Write out our topics
	topics, err := json.Marshal(db.TopicLookup)
	if err != nil {
		return err
	}

	var topicBuffer bytes.Buffer
	w := zlib.NewWriter(&topicBuffer)
	_, err = w.Write(topics)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	tmpPath := filepath.Join(db.Path, "topics.tmp")
	file, err := os.OpenFile(tmpPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(topicBuffer.Bytes())
	if err != nil {
		return err
	}

	err = os.Rename(tmpPath, path.Join(db.Path, "topics"))
	if err != nil {
		return err
	}

	// Write out our topic schemas
	schemas, err := json.Marshal(db.SchemaLookup)
	if err != nil {
		return err
	}

	var schemaBuffer bytes.Buffer
	w = zlib.NewWriter(&schemaBuffer)
	_, err = w.Write(schemas)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	tmpPath = filepath.Join(db.Path, "schemas.tmp")
	file, err = os.OpenFile(tmpPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(schemaBuffer.Bytes())
	if err != nil {
		return err
	}

	err = os.Rename(tmpPath, path.Join(db.Path, "schemas"))
	if err != nil {
		return err
	}

	// Now, write out our metadata
	tmpPath = filepath.Join(db.Path, "metadata.tmp")
	file, err = os.OpenFile(tmpPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(databaseMetadata.Bytes())
	if err != nil {
		return err
	}
	file.Close()

	err = os.Rename(tmpPath, path.Join(db.Path, "metadata"))
	if err != nil {
		return err
	}

	// Next, zero out the WriteAheadLog
	err = os.Remove(filepath.Join(db.Path, "wal.log"))
	if err != nil {
		db.log.Fatal().Err(err).Msg("error removing wal.log")
	}

	// Finally, update our database's STime and appendCount
	db.STime = newSTime
	db.appendCount = 0

	return nil
}

//-- Public Interfaces

func (d *Database) SchemaForTopic(topic string) schema.Object {
	var index int
	var exists bool

	topic = normalizeTopicName(topic)

	d.topicLock.RLock()
	index, exists = d.topics[topic]
	d.topicLock.RUnlock()

	if !exists {
		return nil
	}

	return d.SchemaLookup[index]
}

func (d *Database) AddTopic(topic string, schema string) int {
	topic = normalizeTopicName(topic)

	d.topicLock.RLock()
	if index, exists := d.topics[topic]; exists {
		d.topicLock.RUnlock()
		return index
	}
	d.topicLock.RUnlock()

	// The topic doesn't exist, so get any non-string parent schema
	parentSchema := d.parentSchema(topic)
	// If schema is an empty string, we are doing an implicit topic add,
	// so we should inherit our parent schema
	if parentSchema != nil && schema == "" {
		schema = parentSchema.ToSchema()
	} else if parentSchema != nil && parentSchema.ToSchema() != schema {
		// Otherwise we are trying to create an invalid schema
		// FIXME: This should be an error
		return 0
	}

	// The topic doesn't exist, and the schema is valid, so add it
	d.writeLock.Lock()
	defer d.writeLock.Unlock()

	index := d.addTopicInternal(topic, schema)
	wal := WriteAheadLog{filepath.Join(d.Path, "wal.log")}
	wal.AddTopic(topic, schema)

	return index
}

// Append to the end of the database
func (d *Database) Append(data []byte, topic string) error {
	topicID := d.AddTopic(topic, "")

	s := d.SchemaLookup[topicID]
	if !s.Validate(data) {
		// FIXME: We should either return an error, or move the data to a special topic
		//        when this happens.
		d.log.Error().Msg("Attempted to append non-validating data to a topic")
		return errors.New(fmt.Sprintf("Data does not conform to %s", s.ToSchema()))
	}

	// Explicitly copy the data before taking the lock to minimize resource
	// contention
	e := Datum{Data: make([]byte, len(data)), TopicID: topicID}
	copy(e.Data, data)

	d.writeLock.Lock()
	defer d.writeLock.Unlock()

	if d.appendCount > SegmentSize {
		err := d.serializeInternal()
		if err != nil {
			d.log.Fatal().Msg("Error serializing database to disk.")
		}
	}

	// Pull appendTime now that we have acquired our db lock
	appendTime := time.Now()

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

	return nil
}

func (d *Database) entriesFromData(s *Segment, data []Datum) []Entry {
	entries := make([]Entry, len(data), cap(data))

	for index, val := range data {
		entries[index] = Entry{
			Time:   s.HeadTime.Add(val.Delta),
			Topic:  d.TopicLookup[val.TopicID],
			Schema: d.SchemaLookup[val.TopicID].ToSchema(),
			Data:   val.Data,
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
			startIndex = int(d.Current)
		}
	}

	// If endIndex is 0, that means there are no segments with head times after
	// the specified end time, so use the last segment
	if endIndex == 0 {
		endIndex = int(d.Current)
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
func NewDatabase(name string, location string) (*Database, error) {
	var db Database

	// If the path does not exist, create a new directory
	fileinfo, err := os.Stat(location)
	if os.IsNotExist(err) {
		err := os.Mkdir(location, 0700)
		if err != nil {
			return nil, err
		}
	} else if !fileinfo.IsDir() {
		return nil, fmt.Errorf("supplied path is not a directory")
	}

	// Migrate the database if it's old
	err = MigrateDatabaseIfNeeded(location)
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(filepath.Join(location, "metadata")); err == nil {
		db = Database{
			Path: location,
		}
		err = db.deserializeInternal()
		if err != nil {
			return nil, err
		}
		db.topics = make(map[string]int)
		wal := WriteAheadLog{filepath.Join(db.Path, "wal.log")}
		wal.ApplyToDB(&db)
	} else if _, err = os.Stat(filepath.Join(location, "wal.log")); err == nil {
		db = Database{
			Version:    FossilDBVersion,
			Path:       location,
			Segments:   []Segment{},
			Current:    0,
			topics:     make(map[string]int),
			TopicCount: 0,
		}
		wal := WriteAheadLog{filepath.Join(db.Path, "wal.log")}
		wal.ApplyToDB(&db)
	} else {
		db = Database{
			Version:    FossilDBVersion,
			Path:       location,
			Segments:   []Segment{},
			Current:    0,
			topics:     make(map[string]int),
			TopicCount: 0,
		}
		db.AddTopic("/", "string")
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
		err := db.serializeInternal()
		if err != nil {
			return nil, err
		}
	}
	// Set up our convenience topic map
	for k, v := range db.TopicLookup {
		db.topics[v] = k
	}
	return &db, nil
}
