/*
 * Copyright (c) 2022, Gideon Williams gideon@gideonw.com
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

var (
	// CommandUse sets the current database context
	CommandUse = "USE"
	// CommandError
	CommandError = "ERR"
	// CommandStats retrieves the current database stats
	CommandStats = "STATS"
	// CommandQuery executes a query on the current database
	CommandQuery = "QUERY"
	// CommandAppend appends data to the current database
	CommandAppend = "APPEND"
)
