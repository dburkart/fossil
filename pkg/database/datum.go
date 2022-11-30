/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import (
	"time"
)

type Datum struct {
	Delta   time.Duration
	TopicID int
	Data    []byte
}
