/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package ast

import (
	"fmt"
	"github.com/dburkart/fossil/pkg/common/parse"
	"github.com/dburkart/fossil/pkg/query/scanner"
	"strconv"
	"strings"
	"time"

	"github.com/dburkart/fossil/pkg/database"
)

type ASTNode interface {
	Value() string
}

type Visitor interface {
	Visit(ASTNode) Visitor
}

type FilterGenerator interface {
	GenerateFilter(*database.Database) database.Filter
}

type Numeric interface {
	DerivedValue() int64
}

type (
	BaseNode struct {
		Token parse.Token
	}

	QueryNode struct {
		BaseNode
		Input         string
		Quantifier    ASTNode
		Identifier    ASTNode
		Topic         ASTNode
		TimePredicate ASTNode
		DataPipeline  ASTNode
	}

	QuantifierNode struct {
		BaseNode
		Type         parse.Token
		TimeQuantity ASTNode
	}

	TopicSelectorNode struct {
		BaseNode
		Topic parse.Token
	}

	TimePredicateNode struct {
		BaseNode
		Specifier parse.Token
		Begin     ASTNode
		Comma     parse.Location
		End       ASTNode
	}

	TimeExpressionNode struct {
		BaseNode
		Whence   ASTNode
		Op       parse.Token
		Quantity ASTNode
	}

	TimeWhenceNode struct {
		BaseNode
		When time.Time
	}

	BinaryOpNode struct {
		BaseNode
		Left  ASTNode
		Right ASTNode
	}

	UnaryOpNode struct {
		BaseNode
		Operator parse.Token
		Operand  ASTNode
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
		Elements []ASTNode
	}

	DataPipelineNode struct {
		BaseNode
		Stages []ASTNode
	}

	DataFunctionNode struct {
		BaseNode
		Arguments  []IdentifierNode
		Next       *DataFunctionNode
		Expression ASTNode
	}

	BuiltinFunctionNode struct {
		BaseNode
		Name       parse.Token
		LParen     parse.Location
		Expression ASTNode
		RParen     parse.Location
	}
)

// -- BaseNode

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
			quantity, ok := q.TimeQuantity.(Numeric)
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
	topic := q.Topic.Lexeme

	// Capture the desired topics in our closure
	var topicFilter = make(map[string]bool)

	// Since topics are hierarchical, we want any topic which has the desired prefix
	for _, t := range db.TopicLookup {
		if strings.HasPrefix(t, topic) {
			topicFilter[t] = true
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
		endTime = t.Begin.(*TimeExpressionNode).Time()
		startTime = db.Segments[0].HeadTime
	case "since":
		startTime = t.Begin.(*TimeExpressionNode).Time()
		endTime = time.Now()
	case "between":
		startTime = t.Begin.(*TimeExpressionNode).Time()
		endTime = t.End.(*TimeExpressionNode).Time()
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
	lh := t.Whence.(*TimeWhenceNode)
	tm := lh.Time()

	switch t.Op.Type {
	case scanner.TOK_MINUS:
		rh := t.Quantity.(Numeric)
		return tm.Add(time.Duration(rh.DerivedValue() * -1))
	case scanner.TOK_PLUS:
		rh := t.Quantity.(Numeric)
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
	lh, rh := b.Left.(Numeric), b.Right.(Numeric)

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
