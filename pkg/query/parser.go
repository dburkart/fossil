/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"fmt"
)

type Parser struct {
	Scanner Scanner
}

func (p *Parser) Parse() ASTNode {
	defer func() {
		if e := recover(); e != nil {
			// Print the error
			fmt.Print(e)
		}
	}()
	return p.query()
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

	// TODO: Check for time-predicate
	// TODO: Check for data-predicate

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
		panic(fmt.Sprintf("Error: unexpected token '%s', expected quantifier (all, sample, etc.)", tok.Lexeme))
	}

	q := QuantifierNode{BaseNode{
		Value: tok.Lexeme,
	}}

	if tok.Lexeme == "sample" {
		tok = p.Scanner.Emit()

		if tok.Type != TOK_PAREN_L {
			panic(fmt.Sprintf("Error: unexpected token '%s', expected '('", tok.Lexeme))
		}

		tok = p.Scanner.Emit()

		if tok.Type != TOK_TIMESPAN {
			panic(fmt.Sprintf("Error: unexpected token '%s', expected valid timespan (@hour, @minute, @second, etc.)", tok.Lexeme))
		}
		q.AddChild(&TimespanNode{BaseNode{
			Value: tok.Lexeme,
		}})

		tok = p.Scanner.Emit()

		if tok.Type != TOK_PAREN_R {
			panic(fmt.Sprintf("Error: unexpected token '%s', expected ')'", tok.Lexeme))
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
		panic(fmt.Sprintf("Error: unexpected token '%s', expected a topic after 'in' keyword", tok.Lexeme))
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

	// TODO: Implement + / - time-quantity

	t := TimeExpressionNode{BaseNode{
		Value: "",
	}}
	t.AddChild(whence)

	return &t
}

// timeWhence returns a TimeExpressionNode
//
// Grammar:
//
//	time-whence     = "~now" / "~" RFC3339
func (p *Parser) timeWhence() ASTNode {
	tok := p.Scanner.Emit()

	if tok.Type != TOK_WHENCE {
		panic(fmt.Sprintf("Error: Unexpected token '%s', expected a time-whence (~now, etc.)", tok.Lexeme))
	}

	return &TimeWhenceNode{BaseNode{
		Value: tok.Lexeme,
	}}
}
