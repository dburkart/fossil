/*
 * Copyright (c) 2022, Gideon Williams <gideon@gideonw.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package proto

var (
	// CommandVersion sends the version of the fossil protocol supported to the server / client
	CommandVersion = "VERSION"
	// CommandList sets the current database context
	CommandList = "LIST"
	// CommandUse sets the current database context
	CommandUse = "USE"
	// CommandError sends an error code and a message to the client
	CommandError = "ERR"
	// CommandOk is used to respond to generic actions
	CommandOk = "OK"
	// CommandStats retrieves the current database stats
	CommandStats = "STATS"
	// CommandQuery executes a query on the current database
	CommandQuery = "QUERY"
	// CommandAppend appends data to the current database
	CommandAppend = "APPEND"
	// CommandCreate is used to create topics (but could be used for other purposes in the future)
	CommandCreate = "CREATE"
)
