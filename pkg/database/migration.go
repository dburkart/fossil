/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"encoding/binary"
	"os"
	"path"
)

func DetectVersion(p string) uint32 {
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

func MigrationIsNeeded(p string) bool {
	version := DetectVersion(p)

	if version == 0 {
		return false
	}

	if version < FossilDBVersion {
		return true
	}

	return false
}
