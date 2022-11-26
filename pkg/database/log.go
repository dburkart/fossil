/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	actionAddEvent = 1 << iota
	actionAddSegment
)

type WriteAheadLog struct {
	LogPath string
}

func (w *WriteAheadLog) ApplyToDB(d *Database) {
	file, err := os.OpenFile(w.LogPath, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		action := strings.Split(scanner.Text(), ";")
		actionType, err := strconv.Atoi(action[0])
		if err != nil {
			log.Fatal(err)
		}
		valueBytes, err := base64.StdEncoding.DecodeString(action[1])
		if err != nil {
			log.Fatal(err)
		}

		switch actionType {
		case actionAddEvent:
			var datum Datum
			dec := gob.NewDecoder(bytes.NewBuffer(valueBytes))
			err := dec.Decode(&datum)
			if err != nil {
				log.Fatal(err)
			}
			d.appendInternal(datum)
		case actionAddSegment:
			var segment Segment
			dec := gob.NewDecoder(bytes.NewBuffer(valueBytes))
			err := dec.Decode(&segment.HeadTime)
			if err != nil {
				log.Fatal(err)
			}
			d.Segments = append(d.Segments, segment)
			d.Current += 1
		}
	}
}

func (w *WriteAheadLog) AddEvent(d *Datum) {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(d)
	if err != nil {
		log.Fatal("encode:", err)
	}

	file, err := os.OpenFile(w.LogPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%d;%s\n", actionAddEvent, base64.StdEncoding.EncodeToString(encoded.Bytes())))
	if err != nil {
		log.Fatal(err)
	}
}

func (w *WriteAheadLog) AddSegment(t time.Time) {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(t)
	if err != nil {
		log.Fatal("encode:", err)
	}

	file, err := os.OpenFile(w.LogPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%d;%s\n", actionAddSegment, base64.StdEncoding.EncodeToString(encoded.Bytes())))
	if err != nil {
		log.Fatal(err)
	}
}