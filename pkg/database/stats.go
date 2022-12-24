/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package database

import "time"

type Stats struct {
	Segments      int
	TopicCount    int
	SerializeTime time.Time
}
