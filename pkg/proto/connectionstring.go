/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 * Copyright (c) 2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

import (
	"errors"
	"fmt"
	"net/url"
	"path"
)

var Protocol = "fossil"

type ConnectionString struct {
	Local    bool
	Address  string
	Database string
}

// ParseConnectionString takes a connection string and parses it into the parts
// the application needs to make a connection. This function will always parse,
// even horribly malformed connection strings. It will only return an error if
// the protocol is not "fossil" or "file"
//
// Formats:
//
//	./path/to/local/db
//	file://./path/to/local/db
//	fossil://<host:port>[/<db_name>]
func ParseConnectionString(connStr string) (ConnectionString, error) {
	ret := ConnectionString{
		Local:    true,
		Address:  "local",
		Database: "default",
	}

	if connStr == "" {
		connStr = "./"
	}

	u, err := url.Parse(connStr)
	if err != nil {
		return ConnectionString{}, err
	}

	// Handle the local case
	if u.Scheme == "" || u.Scheme == "file" {
		ret.Database = u.Path
		return ret, nil
	}

	if u.Scheme == "fossil" {
		ret.Local = false
		ret.Address = u.Host
		d, p := path.Split(u.Path)
		if d == "" && p == "" {
			ret.Database = "default"
		} else if d != "/" {
			return ConnectionString{}, errors.New(fmt.Sprintf("invalid database %s", u.Path))
		}
		if p == "" {
			p = "default"
		}
		ret.Database = p
		return ret, nil
	}

	return ConnectionString{}, errors.New(fmt.Sprintf("unrecognized scheme: %s", u.Scheme))
}
