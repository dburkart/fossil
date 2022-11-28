/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query_test

import (
	"fmt"
	"github.com/dburkart/fossil/pkg/query"
	"reflect"
	"testing"
)

func TestParseAllQuantifier(t *testing.T) {
	p := query.Parser{
		Scanner: query.Scanner{
			Input: "all",
		},
	}

	ast := p.Parse()

	if fmt.Sprint(reflect.TypeOf(ast)) != "*query.QueryNode" {
		t.Errorf("wanted root node to be *query.QueryNode, found %s", reflect.TypeOf(ast))
	}

	child := ast.Children()[0]

	if fmt.Sprint(reflect.TypeOf(child)) != "*query.QuantifierNode" {
		t.Errorf("wanted first child to be *query.QuantifierNode, found %s", reflect.TypeOf(child))
	}
}
