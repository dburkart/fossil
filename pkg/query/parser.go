/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type SyntaxError struct {
	Location [2]int
	Message  string
}

func NewSyntaxError(t Token, m string) SyntaxError {
	return SyntaxError{Location: t.Location, Message: m}
}

func (s *SyntaxError) FormatError(input string) string {
	errorString := "Syntax error found in query:\n"
	errorString += input
	errorString += fmt.Sprintf("\n%s^%s ", strings.Repeat(" ", s.Location[0]), strings.Repeat("~", s.Location[1]-s.Location[0]-1))
	errorString += fmt.Sprintf("%s\n", s.Message)
	return errorString
}

type Parser struct {
	Scanner Scanner
}

func (p *Parser) Parse() (query ASTNode, err error) {
	defer func() {
		if e := recover(); e != nil {
			syntaxError, ok := e.(SyntaxError)
			if !ok {
				panic(e)
			}
			err = errors.New(syntaxError.FormatError(p.Scanner.Input))
		}
	}()

	err = nil
	query = p.query()
	return
}

// query returns a QueryNode
//
// Grammar:
//
//	query           = quantifier [ identifier ] [ topic-selector ] [ time-predicate ] [ data-predicate ]
func (p *Parser) query() ASTNode {
	q := QueryNode{BaseNode{
		Value: p.Scanner.Input,
	}}

	// Queries must start with a Quantifier
	q.AddChild(p.quantifier())

	// TODO: Check for identifier

	// Check for topic-selector
	topicSelector := p.topicSelector()
	if topicSelector != nil {
		q.AddChild(topicSelector)
	}

	timePredicate := p.timePredicate()
	if timePredicate != nil {
		q.AddChild(timePredicate)
	}

	// TODO: Check for data-predicate.
	// 		 Note: this will have to be at the beginning of our filters, based
	//	     on the current design

	return &q
}

// quantifier returns a QuantifierNode
//
// Grammar:
//
//	quantifier      = "all" / sample
func (p *Parser) quantifier() ASTNode {
	// Pull off the next token
	tok := p.Scanner.Emit()

	if tok.Type != TOK_KEYWORD || (tok.Lexeme != "all" && tok.Lexeme != "sample") {
		panic(NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected quantifier (all, sample, etc.)", tok.Lexeme)))
	}

	q := QuantifierNode{BaseNode{
		Value: tok.Lexeme,
	}}

	if tok.Lexeme == "sample" {
		tok = p.Scanner.Emit()
		if tok.Type != TOK_PAREN_L {
			panic(NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected '('", tok.Lexeme)))
		}

		q.AddChild(p.timeQuantity())

		tok = p.Scanner.Emit()
		if tok.Type != TOK_PAREN_R {
			panic(NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected ')'", tok.Lexeme)))
		}
	}

	return &q
}

// topicSelector returns a TopicSelectorNode
//
// Grammar:
//
//	topic-selector  = "in" topic
func (p *Parser) topicSelector() ASTNode {
	// Pull off the next token
	tok := p.Scanner.Emit()

	// Ensure it is the "in" keyword
	if tok.Type != TOK_KEYWORD || tok.Lexeme != "in" {
		// topic-selector is optional, so don't error out
		p.Scanner.Rewind()
		return nil
	}

	topic := p.topic()
	t := TopicSelectorNode{BaseNode{
		// TODO: this should be the full in ... selection statement
		Value: "in",
	}}
	t.AddChild(topic)

	return &t
}

// topic returns a TopicNode
//
// Grammar:
//
//	topic           = "/" 1*(ALPHA / DIGIT / "/")
func (p *Parser) topic() ASTNode {
	tok := p.Scanner.Emit()

	if tok.Type != TOK_TOPIC {
		panic(NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected a topic after 'in' keyword", tok.Lexeme)))
	}

	t := TopicNode{BaseNode{
		Value: tok.Lexeme,
	}}

	return &t
}

// timePredicate returns a TimePredicateNode
//
// Grammar:
//
//	time-predicate  = ( "since" time-expression ) / ( "before" time-expression ) /
//	                ( "between" time-expression ".." time-expression )
func (p *Parser) timePredicate() ASTNode {
	tok := p.Scanner.Emit()

	if tok.Type != TOK_KEYWORD || (tok.Lexeme != "since" && tok.Lexeme != "before" &&
		tok.Lexeme != "between") {
		// time-predicates are optional, so don't error out
		p.Scanner.Rewind()
		return nil
	}

	expression := p.timeExpression()

	// TODO: Handle between

	t := TimePredicateNode{BaseNode{
		Value: tok.Lexeme,
	}}
	t.AddChild(expression)

	return &t
}

// timeExpression returns a TimeExpressionNode
//
// Grammar:
//
//	time-expression = ( time-whence ( "-" / "+" ) time-quantity ) / time-whence
func (p *Parser) timeExpression() ASTNode {
	whence := p.timeWhence()

	t := TimeExpressionNode{BaseNode{
		Value: "",
	}}
	t.AddChild(whence)

	tok := p.Scanner.Emit()
	if tok.Lexeme == "-" || tok.Lexeme == "+" {
		t.Value = tok.Lexeme
		t.AddChild(p.timeQuantity())
	} else {
		p.Scanner.Rewind()
	}

	return &t
}

// timeWhence returns a TimeWhenceNode
//
// Grammar:
//
//	time-whence     = "~now" / "~" RFC3339
func (p *Parser) timeWhence() ASTNode {
	tok := p.Scanner.Emit()

	if tok.Type != TOK_WHENCE {
		panic(NewSyntaxError(tok, fmt.Sprintf("Error: Unexpected token '%s', expected a time-whence (~now, etc.)", tok.Lexeme)))
	}

	var when time.Time
	var err error

	switch {
	case tok.Lexeme == "~now":
		when = time.Now()
	case strings.HasPrefix(tok.Lexeme, "~("):
		value := tok.Lexeme[2 : len(tok.Lexeme)-1]
		when, err = time.Parse(time.RFC3339, value)
		if err != nil {
			panic(NewSyntaxError(tok, fmt.Sprintf("Error: Invalid date-time '%s', expected a valid RFC3339 date", value)))
		}
	}

	return &TimeWhenceNode{
		BaseNode: BaseNode{
			Value: tok.Lexeme,
		},
		When: when,
	}
}

// timeQuantity returns either the result of a single timeTerm, or a BinaryOpNode
//
// Grammar:
//
//	time-quantity   = time-term *( ( "-" / "+" ) time-term )
func (p *Parser) timeQuantity() ASTNode {
	lh := p.timeTerm()

	tok := p.Scanner.Emit()
	if tok.Lexeme != "-" && tok.Lexeme != "+" {
		p.Scanner.Rewind()
		return lh
	}

	node := BinaryOpNode{BaseNode{
		Value: tok.Lexeme,
	}}

	rh := p.timeTerm()

	node.AddChild(lh)
	node.AddChild(rh)
	return &node
}

// timeTerm returns the result of a timeAtom, or a BinaryOpNode
//
// Grammar:
//
//	time-term       = time-atom *( "*" time-atom )
func (p *Parser) timeTerm() ASTNode {
	lh := p.timeAtom()

	tok := p.Scanner.Emit()
	if tok.Lexeme != "*" {
		p.Scanner.Rewind()
		return lh
	}

	node := BinaryOpNode{BaseNode{
		Value: tok.Lexeme,
	}}

	rh := p.timeAtom()

	node.AddChild(lh)
	node.AddChild(rh)
	return &node
}

// timeAtom returns a NumberNode, or a TimespanNode
//
// Grammar:
//
//	time-atom       = number / timespan
func (p *Parser) timeAtom() ASTNode {
	tok := p.Scanner.Emit()

	switch tok.Type {
	case TOK_NUMBER:
		return &NumberNode{BaseNode{
			Value: tok.Lexeme,
		}}
	case TOK_TIMESPAN:
		return &TimespanNode{BaseNode{
			Value: tok.Lexeme,
		}}
	}

	panic(NewSyntaxError(tok, fmt.Sprintf("Expected number of timespan, got '%s'", tok.Lexeme)))
}
