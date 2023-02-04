/*
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package fossil

import (
	"github.com/dburkart/fossil/pkg/database"
	"github.com/dburkart/fossil/pkg/proto"
)

type Client interface {
	Close() error
	Send(proto.Message) (proto.Message, error)
	Append(string, []byte) error
	Query(string) (database.Entries, error)
}
