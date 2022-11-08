/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

const (
	actionAddEvent = 1 << iota
)

type WriteAheadLog struct {
	LogPath string
}

func (w *WriteAheadLog) AddEvent(e *Event) {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(e)
	if err != nil {
		log.Fatal("encode:", err)
	}

	file, err := os.OpenFile(w.LogPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%d\n%s\n", actionAddEvent, base64.StdEncoding.EncodeToString(encoded.Bytes())))
	if err != nil {
		log.Fatal(err)
	}
}
