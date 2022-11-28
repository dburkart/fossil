/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseAllQuantifier(t *testing.T) {
	p := Parser{
		Scanner: Scanner{
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

	filters := ast.Walk(nil)
	if len(filters) != 1 {
		t.Errorf("wanted 1 filter, got %d instead", len(filters))
	}
}

func TestParseTopicSelector(t *testing.T) {
	p := Parser{
		Scanner: Scanner{
			Input: "in /foo/bar/baz",
		},
	}

	ast := p.topicSelector()
	if fmt.Sprint(reflect.TypeOf(ast)) != "*query.TopicSelectorNode" {
		t.Errorf("wanted root node to be *query.TopicSelectorNode, found %s", reflect.TypeOf(ast))
	}

	child := ast.Children()[0]
	if fmt.Sprint(reflect.TypeOf(child)) != "*query.TopicNode" {
		t.Errorf("wanted first child to be *query.TopicNode, found %s", reflect.TypeOf(child))
	}
}
