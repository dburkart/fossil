/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import (
	"errors"
	"fmt"
	"github.com/dburkart/fossil/pkg/common/parse"
	"strings"
	"time"
)

type Parser struct {
	Scanner Scanner
}

func (p *Parser) Parse() (query ASTNode, err error) {
	defer func() {
		if e := recover(); e != nil {
			syntaxError, ok := e.(parse.SyntaxError)
			if !ok {
				panic(e)
			}
			err = errors.New(syntaxError.FormatError(p.Scanner.Input))
		}
	}()

	// Now that we don't allow valid input after a valid query, make sure to
	// trim queries of whitespace
	p.Scanner.Input = strings.Trim(p.Scanner.Input, " \t\n")

	err = nil
	query = p.query()

	// If we didn't parse all the input, return an error
	if p.Scanner.Pos != len(p.Scanner.Input) {
		syntaxError := parse.NewSyntaxError(parse.Token{
			Type:     TOK_INVALID,
			Location: [2]int{p.Scanner.Pos, len(p.Scanner.Input) - 1},
		}, "Error: query is not valid, starting here")
		err = errors.New(syntaxError.FormatError(p.Scanner.Input))
	}

	return
}

// query returns a QueryNode
//
// Grammar:
//
//	query           = quantifier [ topic-selector ] [ time-predicate ] [ data-predicate ] [ data-pipeline ]
func (p *Parser) query() ASTNode {
	q := QueryNode{BaseNode{
		Value: p.Scanner.Input,
	}}

	// Queries must start with a Quantifier
	q.AddChild(p.quantifier())

	// Check for topic-selector
	topicSelector := p.topicSelector()
	if topicSelector != nil {
		q.AddChild(topicSelector)
	}

	timePredicate := p.timePredicate()
	if timePredicate != nil {
		q.AddChild(timePredicate)
	}

	dataPipeline := p.dataPipeline()
	if dataPipeline != nil {
		q.AddChild(dataPipeline)
	}

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
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected quantifier (all, sample, etc.)", tok.Lexeme)))
	}

	q := QuantifierNode{BaseNode{
		Value: tok.Lexeme,
	}}

	if tok.Lexeme == "sample" {
		tok = p.Scanner.Emit()
		if tok.Type != TOK_PAREN_L {
			panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected '('", tok.Lexeme)))
		}

		q.AddChild(p.timeQuantity())

		tok = p.Scanner.Emit()
		if tok.Type != TOK_PAREN_R {
			panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected ')'", tok.Lexeme)))
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

	if tok.Type != TOK_TOPIC && tok.Type != TOK_SLASH {
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected a topic after 'in' keyword", tok.Lexeme)))
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

	lh := p.timeExpression()

	t := TimePredicateNode{BaseNode{
		Value: tok.Lexeme,
	}}
	t.AddChild(lh)

	if tok.Lexeme == "between" {
		comma := p.Scanner.Emit()

		if comma.Lexeme != "," {
			panic(parse.NewSyntaxError(comma, fmt.Sprintf("Error: unexpected token '%s', expected ','", comma.Lexeme)))
		}

		rh := p.timeExpression()
		t.AddChild(rh)
	}

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
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: Unexpected token '%s', expected a time-whence (~now, etc.)", tok.Lexeme)))
	}

	var when time.Time
	var err error

	switch {
	case tok.Lexeme == "~now":
		when = time.Now()
	case strings.HasPrefix(tok.Lexeme, "~("):
		value := tok.Lexeme[2 : len(tok.Lexeme)-1]
		when, err = ParseVagueDateTime(value)
		if err != nil {
			panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: %s", err.Error())))
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

	panic(parse.NewSyntaxError(tok, fmt.Sprintf("Expected number of timespan, got '%s'", tok.Lexeme)))
}

// dataPipeline returns a DataPipelineNode, or nil
//
// Grammar:
//
//	data-pipeline   = 1*data-stage
func (p *Parser) dataPipeline() ASTNode {
	stage := p.dataStage()
	if stage == nil {
		return nil
	}

	stages := []ASTNode{}

	for stage != nil {
		stages = append(stages, stage)
		stage = p.dataStage()
	}

	return &DataPipelineNode{BaseNode{children: stages}}
}

// dataStage returns a DataFunctionNode, or nil if it's not a stage
//
// Grammar:
//
//	data-stage      = ":" data-function
func (p *Parser) dataStage() ASTNode {
	t := p.Scanner.Emit()
	if t.Type != TOK_COLON {
		p.Scanner.Rewind()
		return nil
	}

	return p.dataFunction()
}

// dataFunction returns a DataFunctionNode, or errors out
//
// Grammar:
//
//	data-function   = ( "filter" / "map" / "reduce" ) data-args "->" ( expression / tuple )
//	data-args       = identifier [ "," data-args ]
func (p *Parser) dataFunction() ASTNode {
	t := p.Scanner.Emit()
	if t.Type != TOK_KEYWORD && t.Lexeme != "map" && t.Lexeme != "reduce" &&
		t.Lexeme != "filter" {
		panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Unexpected token '%s', expected 'filter', 'map', or 'reduce'", t.Lexeme)))
	}

	fn := DataFunctionNode{BaseNode: BaseNode{Value: t.Lexeme}}

	// First, parse arguments
	t = p.Scanner.Emit()
	for {
		// We're done parsing arguments when we hit a '->'
		if t.Type == TOK_ARROW {
			break
		}

		if t.Type != TOK_IDENTIFIER {
			panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Unexpected token '%s', expected an identifier", t.Lexeme)))
		}

		fn.Arguments = append(fn.Arguments, IdentifierNode{BaseNode: BaseNode{Value: t.Lexeme}})

		// Pull off a comma if one exists
		t = p.Scanner.Emit()

		if t.Type != TOK_COMMA {
			continue
		}

		// Next argument
		t = p.Scanner.Emit()
	}

	fn.children = append(fn.children, p.tuple())

	return &fn
}

// expression returns a BinaryOpNode, or the result of comparison
//
// Grammar:
//
//	expression      = comparison *( ( "!=" / "==" ) expression )
func (p *Parser) expression() ASTNode {
	c := p.comparison()

	t := p.Scanner.Emit()
	if t.Type == TOK_NOT_EQ || t.Type == TOK_EQ_EQ {
		op := BinaryOpNode{BaseNode{Value: t.Lexeme}}
		op.children = append(op.children, c)
		op.children = append(op.children, p.expression())
		return &op
	}
	p.Scanner.Rewind()

	return c
}

// comparison returns a BinaryOpNode, or the result of term
//
// Grammar:
//
//	comparison      = term *( ( ">" / ">=" / "<" / "<=" ) comparison )
func (p *Parser) comparison() ASTNode {
	t := p.term()

	c := p.Scanner.Emit()
	if c.Type == TOK_GREATER || c.Type == TOK_GREATER_EQ ||
		c.Type == TOK_LESS || c.Type == TOK_LESS_EQ {
		op := BinaryOpNode{BaseNode{Value: c.Lexeme}}
		op.children = append(op.children, t)
		op.children = append(op.children, p.comparison())
		return &op
	}
	p.Scanner.Rewind()

	return t
}

// term returns a BinaryOpNode, or the result of term_md
//
// Grammar:
//
//	term            = term_md *( ( "-" / "+" ) term )
func (p *Parser) term() ASTNode {
	t := p.termMD()

	c := p.Scanner.Emit()
	if c.Type == TOK_MINUS || c.Type == TOK_PLUS {
		op := BinaryOpNode{BaseNode{Value: c.Lexeme}}
		op.children = append(op.children, t)
		op.children = append(op.children, p.term())
		return &op
	}
	p.Scanner.Rewind()

	return t
}

// termMD returns a BinaryOpNode, or the result of unary
//
// Grammar:
//
//	term_md         = unary *( ( "/" / "*" ) term_md )
func (p *Parser) termMD() ASTNode {
	u := p.unary()

	c := p.Scanner.Emit()
	if c.Type == TOK_SLASH || c.Type == TOK_STAR {
		op := BinaryOpNode{BaseNode{Value: c.Lexeme}}
		op.children = append(op.children, u)
		op.children = append(op.children, p.termMD())
		return &op
	}
	p.Scanner.Rewind()

	return u
}

// unary returns a UnaryOpNode, or the result of primary
//
// Grammar:
//
//	unary           = ( "-" / "+" ) ( number / identifier ) / primary
func (p *Parser) unary() ASTNode {
	t := p.Scanner.Emit()
	if t.Type == TOK_MINUS || t.Type == TOK_PLUS {
		op := UnaryOpNode{BaseNode{Value: t.Lexeme}}
		t = p.Scanner.Emit()

		if t.Type == TOK_NUMBER {
			op.children = append(op.children, &NumberNode{BaseNode{Value: t.Lexeme}})
		} else if t.Type == TOK_IDENTIFIER {
			op.children = append(op.children, &IdentifierNode{BaseNode{Value: t.Lexeme}})
		} else {
			panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Unexpected token '%s'. Expected a number or identifier.", t.Lexeme)))
		}

		return &op
	}
	p.Scanner.Rewind()

	return p.primary()
}

// primary returns a leaf node for an expression
//
// Grammar:
//
//	primary         = identifier / number / string / tuple / builtin
func (p *Parser) primary() ASTNode {
	t := p.Scanner.Emit()

	switch t.Type {
	case TOK_IDENTIFIER:
		return &IdentifierNode{BaseNode{Value: t.Lexeme}}
	case TOK_NUMBER:
		return &NumberNode{BaseNode{Value: t.Lexeme}}
	case TOK_STRING:
		return &StringNode{BaseNode{Value: t.Lexeme}}
	default:
		panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Unexpected token '%s'. Expected identifier, number, string, tuple or builtin.", t.Lexeme)))
	}
}

// tuple returns a TupleNode or nil
//
// Grammar:
//
//	tuple           = expression 1*( "," expression )
func (p *Parser) tuple() ASTNode {
	list := TupleNode{}
	e := p.expression()
	t := p.Scanner.Emit()
	list.children = append(list.children, e)

	for t.Type == TOK_COMMA {
		e = p.expression()
		list.children = append(list.children, e)
		t = p.Scanner.Emit()
	}

	p.Scanner.Rewind()

	// We weren't a tuple, so restore scanner, and return nil
	if len(list.children) == 1 {
		return e
	}

	return &list
}
