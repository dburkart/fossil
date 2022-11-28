/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import "github.com/dburkart/fossil/pkg/database"

func Prepare(d *database.Database, statement string) []database.Filter {
	p := Parser{
		Scanner{
			Input: statement,
		},
	}

	ast := p.Parse()

	if ast == nil {
		return []database.Filter{}
	}

	// Walk the tree
	filters := ast.Walk(d)

	return filters
}
