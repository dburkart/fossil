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
	"github.com/dburkart/fossil/pkg/query/types"
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
		In    parse.Location
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
		Op    parse.Token
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
		Val types.Value
	}

	StringNode struct {
		BaseNode
		Val types.Value
	}

	TupleNode struct {
		BaseNode
		Elements []ASTNode
	}

	ElementNode struct {
		BaseNode
		Identifier IdentifierNode
		Subscript  ASTNode
	}

	DataPipelineNode struct {
		BaseNode
		Stages []ASTNode
	}

	DataFunctionNode struct {
		BaseNode
		Name       parse.Token
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

	CompositeNode struct {
		BaseNode
		Keys   []StringNode
		Values []ASTNode
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

//-- TopicSelectorNode

func (t TopicSelectorNode) Value() string {
	return "in"
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
	case "@week":
		return int64(time.Hour * 24 * 7)
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

//-- StringNode

func MakeStringNode(tok parse.Token) *StringNode {
	return &StringNode{BaseNode: BaseNode{Token: tok}, Val: types.MakeFromToken(tok)}
}

// MakeStringNodeFromID is syntactic sugar to allow unquoted strings in some instances
func MakeStringNodeFromID(tok parse.Token) *StringNode {
	return &StringNode{BaseNode: BaseNode{Token: tok}, Val: types.MakeString(tok.Lexeme)}
}

//-- NumberNode

func MakeNumberNode(tok parse.Token) *NumberNode {
	return &NumberNode{BaseNode: BaseNode{Token: tok}, Val: types.MakeFromToken(tok)}
}

func (n NumberNode) DerivedValue() int64 {
	return types.IntVal(n.Val)
}
