/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"fmt"
	"time"
)

type Datum struct {
	Delta   time.Duration
	TopicID int
	Data    []byte
}

func (d *Datum) ToString() string {
	return fmt.Sprintf("%d\t%s", d.TopicID, string(d.Data))
}

type Filter func([]Datum) []Datum
