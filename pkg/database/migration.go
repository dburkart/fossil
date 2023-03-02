/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"os"
	"path"
	"time"
)

// deserializeFunc reads in a database of a specific version, and returns a
// versioned database object
type deserializeFunc func(string) (any, error)

// migrationFunc takes a database of a specific version, and returns one of
// another version
type migrationFunc func(any) (any, error)

// cleanupFunc cleans up a database path for a specific version
type cleanupFunc func(string) error

var deserializationFunctions = []deserializeFunc{
	nil,
	deserializeV1,
}

var migrationFunctions = []migrationFunc{
	nil,
	migrateV1ToV2,
}

var cleanupFunctions = []cleanupFunc{
	nil,
	cleanupV1,
}

//--
//-- Database Version 1 migration handlers
//--

type databaseV1 struct {
	Version     int
	Name        string
	Path        string
	Segments    []Segment
	Current     int
	TopicLookup []string
	TopicCount  int
}

func deserializeV1(p string) (any, error) {
	file, err := os.ReadFile(path.Join(p, "database"))
	if err != nil {
		return nil, err
	}

	var db databaseV1
	dec := gob.NewDecoder(bytes.NewBuffer(file))
	err = dec.Decode(&db)
	if err != nil {
		return nil, err
	}

	// Never trust the Path field in the database
	db.Path = p

	return &db, nil
}

func migrateV1ToV2(db any) (any, error) {
	// Assert that from is a v1 database
	from, ok := db.(*databaseV1)
	if !ok {
		return nil, errors.New("attempted migration from a non v1 database")
	}

	to := Database{
		Version:     2,
		Segments:    from.Segments,
		Current:     uint32(from.Current),
		TopicLookup: from.TopicLookup,
		TopicCount:  from.TopicCount,
		STime:       time.Now(),
		Path:        from.Path,
		Name:        from.Name,
	}

	defaultSchema := to.loadSchema("string")
	for range to.TopicLookup {
		to.SchemaLookup = append(to.SchemaLookup, defaultSchema)
	}

	return &to, nil
}

func cleanupV1(p string) error {
	// Remove any "database" file
	err := os.Remove(path.Join(p, "database"))
	if err != nil {
		return err
	}
	return nil
}

// The detectVersion function is responsible for detecting the version for a given
// on-disk database. Starting with version 2, the version will always be stored as
// the first 4 bytes of the database's metadata file. We special case version 1
// since the format is completely different there.
//
// If this function returns a version of "0", that indicates that we think this is
// a "version-less" database; i.e. the database has never spilled to disk, and only
// a write-ahead log exists.
func detectVersion(p string) uint32 {
	// Versions without a metadata file that have a database file are version 1
	if _, err := os.Stat(path.Join(p, "metadata")); os.IsNotExist(err) {
		if _, err := os.Stat(path.Join(p, "database")); !os.IsNotExist(err) {
			return 1
		}

		// If metadata simply does not exist yet, we are "version-less", and
		// data must only exist in our write-ahead-log
		return 0
	}

	// If we have a database file, read in the version
	file, err := os.Open(path.Join(p, "metadata"))
	if err != nil {
		return 0
	}
	defer file.Close()

	var version uint32
	err = binary.Read(file, binary.LittleEndian, &version)
	if err != nil {
		return 0
	}

	return version
}

// migrationIsNeeded returns whether we think a migration is necessary.
// This function bases the decision on whether the detected version is less
// than FossilDBVersion (excluding 0, which indicates we are version-less).
func migrationIsNeeded(p string) bool {
	version := detectVersion(p)

	if version == 0 {
		return false
	}

	if version < FossilDBVersion {
		return true
	}

	return false
}

// MigrateDatabaseIfNeeded has all the logic necessary to migrate from
// an old database through an arbitrary number of versions to the current
// database version. It does this through use of 3 types of handler functions:
//
//   - A deserialization function, specific to a source DB version.
//     This function is responsible for reading the on-disk structure of
//     a specific database format.
//
//   - Some number of migration functions. Migration functions map from
//     version n - 1 -> n, so each function will be called in succession,
//     passing the results of the previous migration.
//
//   - A cleanup function, specific to a source DB version. This function
//     is responsible for cleaning up any un-needed files after migration.
func MigrateDatabaseIfNeeded(p string) error {
	if !migrationIsNeeded(p) {
		return nil
	}

	dbVersion := detectVersion(p)

	// First deserialize the old database
	db, err := deserializationFunctions[dbVersion](p)
	if err != nil {
		return err
	}

	// Now, migrate the database struct
	for _, m := range migrationFunctions[dbVersion:] {
		if m == nil {
			continue
		}

		db, err = m(db)
		if err != nil {
			return err
		}
	}

	// We must have a Database struct at the end of the migration
	modernDB, ok := db.(*Database)
	if !ok {
		return errors.New("expected a Database at the end of migration")
	}

	// Serialize the migrated DB to disk
	err = modernDB.serializeInternal()
	if err != nil {
		return err
	}

	// Now, perform cleanup
	cleanup := cleanupFunctions[dbVersion]
	if cleanup != nil {
		err = cleanup(p)
		if err != nil {
			return err
		}
	}

	return nil
}
