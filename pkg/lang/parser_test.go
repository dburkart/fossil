/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package lang_test

import (
	"fmt"
	"github.com/dburkart/fossil/pkg/lang"
	"reflect"
	"testing"
)

func TestParseAllQuantifier(t *testing.T) {
	p := lang.Parser{
		Scanner: lang.Scanner{
			Input: "all",
		},
	}

	ast := p.Parse()

	if fmt.Sprint(reflect.TypeOf(ast)) != "*lang.QueryNode" {
		t.Errorf("wanted root node to be *lang.QueryNode, found %s", reflect.TypeOf(ast))
	}

	child := ast.Children()[0]

	if fmt.Sprint(reflect.TypeOf(child)) != "*lang.QuantifierNode" {
		t.Errorf("wanted first child to be *lang.QuantifierNode, found %s", reflect.TypeOf(child))
	}
}
