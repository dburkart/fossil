/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"fmt"
	"github.com/dburkart/fossil/pkg/common/parse"
	"github.com/dburkart/fossil/pkg/schema"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/dburkart/fossil/pkg/database"
)

func (b *BaseNode) ToString() string {
	t := reflect.TypeOf(b)
	return t.Elem().Name()
}

func ASTToString(ast ASTNode) string {
	return ASTToStringInternal(ast, 0)
}

func ASTToStringInternal(ast ASTNode, indent int) string {
	level := strings.Repeat("    ", indent)

	value := ast.Value()
	switch t := ast.(type) {
	case *DataFunctionNode:
		var args string
		for _, a := range t.Arguments {
			args += a.Value() + ", "
		}
		value = "name(" + ast.Value() + ") args(" + args[:len(args)-2] + ")"
	}

	t := reflect.TypeOf(ast)
	output := level + t.Elem().Name() + "[" + value + "]" + "\n"

	for _, child := range ast.Children() {
		output += ASTToStringInternal(child, indent+1)
	}

	return output
}

type ASTNode interface {
	Children() []ASTNode
	Walk(*database.Database) []database.Filter
	Value() string
	Type() schema.Object
}

type Visitor interface {
	Visit(ASTNode) error
}

type FilterGenerator interface {
	GenerateFilter(*database.Database) database.Filter
}

type Numeric interface {
	DerivedValue() int64
}

func WalkTree(root ASTNode, v Visitor) error {
	if len(root.Children()) == 0 {
		err := v.Visit(root)
		if err != nil {
			return err
		}
		return nil
	}

	for _, child := range root.Children() {
		err := WalkTree(child, v)
		if err != nil {
			return err
		}
	}

	err := v.Visit(root)
	if err != nil {
		return err
	}

	return nil
}

type (
	BaseNode struct {
		Token    parse.Token
		TypeI    schema.Object
		children []ASTNode
	}

	QueryNode struct {
		BaseNode
		Input string
	}

	QuantifierNode struct {
		BaseNode
	}

	TopicSelectorNode struct {
		BaseNode
	}

	TopicNode struct {
		BaseNode
	}

	TimePredicateNode struct {
		BaseNode
	}

	TimeExpressionNode struct {
		BaseNode
	}

	TimeWhenceNode struct {
		BaseNode
		When time.Time
	}

	BinaryOpNode struct {
		BaseNode
	}

	UnaryOpNode struct {
		BaseNode
	}

	TimespanNode struct {
		BaseNode
	}

	IdentifierNode struct {
		BaseNode
	}

	NumberNode struct {
		BaseNode
	}

	StringNode struct {
		BaseNode
	}

	TupleNode struct {
		BaseNode
	}

	DataPipelineNode struct {
		BaseNode
	}

	DataFunctionNode struct {
		BaseNode
		Arguments []IdentifierNode
		Next      *DataFunctionNode
	}

	BuiltinFunctionNode struct {
		BaseNode
	}
)

//-- BaseNode

func (b *BaseNode) Children() []ASTNode {
	return b.children
}

func (b *BaseNode) AddChild(child ASTNode) {
	b.children = append(b.children, child)
}

func (b *BaseNode) Type() schema.Object {
	return b.TypeI
}

func (b *BaseNode) descend(d *database.Database, n ASTNode) []database.Filter {
	var t FilterGenerator
	var isFilter bool
	var f database.Filter

	if t, isFilter = n.(FilterGenerator); isFilter {
		f = t.GenerateFilter(d)
	}

	if len(n.Children()) == 0 {
		if isFilter {
			return []database.Filter{f}
		} else {
			return []database.Filter{}
		}
	}

	var chain []database.Filter

	if f != nil {
		chain = append(chain, f)
	}

	for _, child := range n.Children() {
		chain = append(chain, b.descend(d, child)...)
	}

	return chain
}

func (b *BaseNode) Walk(d *database.Database) []database.Filter {
	return b.descend(d, b)
}

func (b *BaseNode) Value() string {
	return b.Token.Lexeme
}

//-- QueryNode

func (q QueryNode) Value() string {
	return q.Input
}

func (q QueryNode) GenerateFilter(_ *database.Database) database.Filter {
	return nil
}

//-- QuantifierNode

func (q QuantifierNode) GenerateFilter(db *database.Database) database.Filter {
	return func(data database.Entries) database.Entries {
		if data == nil {
			data = db.Retrieve(database.Query{Quantifier: q.Value(), Range: nil})
		}

		switch q.Value() {
		case "all":
			return data
		case "sample":
			quantity, ok := q.Children()[0].(Numeric)
			if !ok {
				panic("Expected child to be of type *TimespanNode")
			}

			sampleDuration := quantity.DerivedValue()
			nextTime := data[0].Time
			filtered := database.Entries{}

			for _, val := range data {
				if val.Time.After(nextTime) || val.Time.Equal(nextTime) {
					filtered = append(filtered, val)
					nextTime = val.Time.Add(time.Duration(sampleDuration))
				}
			}

			return filtered

		}
		// TODO: What's the right thing to return here? Maybe we should panic?
		return database.Entries{}
	}
}

//-- TopicSelectorNode

func (t TopicSelectorNode) Value() string {
	return "in"
}

func (q TopicSelectorNode) GenerateFilter(db *database.Database) database.Filter {
	topic, ok := q.Children()[0].(*TopicNode)
	if !ok {
		panic("Expected child to be of type *TopicNode")
	}
	topicName := topic.Value()

	// Capture the desired topics in our closure
	var topicFilter = make(map[string]bool)

	// Since topics are hierarchical, we want any topic which has the desired prefix
	for _, topic := range db.TopicLookup {
		if strings.HasPrefix(topic, topicName) {
			topicFilter[topic] = true
		}
	}

	return func(data database.Entries) database.Entries {
		if data == nil {
			data = db.Retrieve(database.Query{Range: nil})
		}

		filtered := database.Entries{}

		for _, val := range data {
			if _, ok := topicFilter[val.Topic]; ok {
				filtered = append(filtered, val)
			}
		}

		return filtered
	}
}

//-- TimePredicateNode

func (t TimePredicateNode) GenerateFilter(db *database.Database) database.Filter {
	var startTime, endTime time.Time

	switch t.Value() {
	case "before":
		endTime = t.Children()[0].(*TimeExpressionNode).Time()
		startTime = db.Segments[0].HeadTime
	case "since":
		startTime = t.Children()[0].(*TimeExpressionNode).Time()
		endTime = time.Now()
	case "between":
		startTime = t.Children()[0].(*TimeExpressionNode).Time()
		endTime = t.Children()[1].(*TimeExpressionNode).Time()
	}

	timeRange := database.TimeRange{Start: startTime, End: endTime}

	return func(data database.Entries) database.Entries {
		if data == nil {
			return db.Retrieve(database.Query{Range: &timeRange, RangeSemantics: t.Value()})
		}

		// TODO: Handle non-nil case! Let's factor out some of the Retrieve functionality for
		//       filtering ranges.
		return nil
	}
}

//-- TimeExpressionNode

func (t TimeExpressionNode) Time() time.Time {
	lh := t.Children()[0].(*TimeWhenceNode)
	tm := lh.Time()

	switch t.Value() {
	case "-":
		rh := t.Children()[1].(Numeric)
		return tm.Add(time.Duration(rh.DerivedValue() * -1))
	case "+":
		rh := t.Children()[1].(Numeric)
		return tm.Add(time.Duration(rh.DerivedValue()))
	}

	return tm
}

//-- TimeWhenceNode

func (t TimeWhenceNode) Time() time.Time {
	return t.When
}

//-- BinaryOpNode

func (b BinaryOpNode) DerivedValue() int64 {
	lh, rh := b.Children()[0].(Numeric), b.Children()[1].(Numeric)

	switch b.Value() {
	case "*":
		return lh.DerivedValue() * rh.DerivedValue()
	case "-":
		return lh.DerivedValue() - rh.DerivedValue()
	case "+":
		return lh.DerivedValue() + rh.DerivedValue()
	}

	panic(fmt.Sprintf("Unknown operator '%s'", b.Value()))
}

//-- TimespanNode

func (t TimespanNode) DerivedValue() int64 {
	switch t.Value() {
	case "@year":
		return int64(time.Hour * 24 * 365)
	case "@month":
		return int64(time.Hour * 24 * 30)
	case "@day":
		return int64(time.Hour * 24)
	case "@hour":
		return int64(time.Hour)
	case "@minute":
		return int64(time.Minute)
	case "@second":
		return int64(time.Second)
	}
	return 0
}

//-- NumberNode

func (n NumberNode) DerivedValue() int64 {
	i, err := strconv.ParseInt(n.Value(), 10, 64)
	if err != nil {
		panic(fmt.Sprintf("NumberNode had unexpected non-numerical value: %s", n.Value()))
	}
	return i
}
