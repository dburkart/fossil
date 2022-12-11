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
	"time"
)

func TestGarbageAfterQuery(t *testing.T) {
	p := Parser{
		Scanner: Scanner{
			Input: "all and then some garbage",
		},
	}

	_, err := p.Parse()
	if err == nil {
		t.Fail()
	}

	p = Parser{
		Scanner: Scanner{
			Input: "all  \t\n",
		},
	}

	_, err = p.Parse()
	if err != nil {
		t.Fail()
	}
}

func TestEmptyQuery(t *testing.T) {
	p := Parser{
		Scanner: Scanner{
			Input: "",
		},
	}

	_, err := p.Parse()
	if err == nil {
		t.Fail()
	}
}

func TestParseAllQuantifier(t *testing.T) {
	p := Parser{
		Scanner: Scanner{
			Input: "all",
		},
	}

	ast, _ := p.Parse()

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

func TestParseTimePredicate(t *testing.T) {
	p := Parser{
		Scanner: Scanner{
			Input: "since ~now",
		},
	}

	ast := p.timePredicate()
	if fmt.Sprint(reflect.TypeOf(ast)) != "*query.TimePredicateNode" {
		t.Errorf("wanted root node to be *query.TimePredicateNode, found %s", reflect.TypeOf(ast))
	}

	child := ast.Children()[0]
	if fmt.Sprint(reflect.TypeOf(child)) != "*query.TimeExpressionNode" {
		t.Errorf("wanted first child to be *query.TimeExpressionNode, found %s", reflect.TypeOf(child))
	}

	child = child.Children()[0]
	if fmt.Sprint(reflect.TypeOf(child)) != "*query.TimeWhenceNode" {
		t.Errorf("wanted first child to be *query.TimeWhenceNode, found %s", reflect.TypeOf(child))
	}
}

func TestTimeWhence(t *testing.T) {
	p := Parser{
		Scanner: Scanner{
			Input: "~(1996-12-19T16:39:57-08:00)",
		},
	}

	ast := p.timeWhence()
	if fmt.Sprint(reflect.TypeOf(ast)) != "*query.TimeWhenceNode" {
		t.Errorf("wanted first child to be *query.TimeWhenceNode, found %s", reflect.TypeOf(ast))
	}

	want, _ := time.Parse(time.RFC3339, "1996-12-19T16:39:57-08:00")

	tm := ast.(*TimeWhenceNode).Time()
	if !tm.Equal(want) {
		t.Errorf("wanted time-whence to parse to %s, got %s", want, tm)
	}
}
