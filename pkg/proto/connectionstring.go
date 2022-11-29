package proto

import (
	"fmt"
	"strings"
)

var Protocol = "fossil"

type ConnectionString struct {
	Local    bool
	Address  string
	Database string
}

// ParseConnectionString takes a connection string and parses it into the parts
// the application needs to make a connection. This function will always parse,
// even horribly malformed connection strings. It will only panic if a protocol
// is specified and it is != 'fossil'.
// Format:
//
//	[fossil://]<host:port|local>[/<db_name>]
func ParseConnectionString(connStr string) ConnectionString {
	ret := ConnectionString{
		Local:    true,
		Address:  "local",
		Database: "default",
	}

	// if there is no connStr, use local and default
	if len(connStr) == 0 {
		return ret
	}

	protoSep := strings.Index(connStr, "://")
	if protoSep != -1 {
		if connStr[:protoSep] != Protocol {
			panic(fmt.Sprintf("Unsupported protocol '%s'. ", connStr[:protoSep]))
		}
	}

	// Remove the optional protocol prefix
	connStr = strings.TrimPrefix(connStr, Protocol+"://")

	// strip ending slash before assigning values
	connStr = strings.TrimSuffix(connStr, "/")

	// if there is no connStr, after removing all parts, use local and default
	if len(connStr) == 0 {
		return ret
	}

	// then search for path separator
	delim := strings.Index(connStr, "/")
	if delim == -1 {
		ret.Address = connStr
		ret.Database = "default"
	} else {
		ret.Address = connStr[:delim]
		ret.Database = connStr[delim+1:]
	}

	if ret.Address != "local" {
		ret.Local = false
	}

	return ret
}
