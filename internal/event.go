/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package internal

import "time"

type Event struct {
	Timestamp time.Time
	Data      string
}
