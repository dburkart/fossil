/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package parser

import (
	"errors"
	"fmt"
	"github.com/dburkart/fossil/pkg/common/parse"
	"github.com/dburkart/fossil/pkg/query/ast"
	"github.com/dburkart/fossil/pkg/query/scanner"
	"strings"
	"time"
)

type Parser struct {
	Scanner scanner.Scanner
}

func (p *Parser) Parse() (query ast.ASTNode, err error) {
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
			Type:     scanner.TOK_INVALID,
			Location: parse.Location{Start: p.Scanner.Pos, End: len(p.Scanner.Input) - 1},
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
func (p *Parser) query() ast.ASTNode {
	q := ast.QueryNode{BaseNode: ast.BaseNode{}, Input: p.Scanner.Input}

	// Queries must start with a Quantifier
	q.Quantifier = p.quantifier()

	// Check for topic-selector
	topicSelector := p.topicSelector()
	if topicSelector != nil {
		q.Topic = topicSelector
	}

	timePredicate := p.timePredicate()
	if timePredicate != nil {
		q.TimePredicate = timePredicate
	}

	dataPipeline := p.dataPipeline()
	if dataPipeline != nil {
		q.DataPipeline = dataPipeline
	}

	return &q
}

// quantifier returns a QuantifierNode
//
// Grammar:
//
//	quantifier      = "all" / sample
func (p *Parser) quantifier() ast.ASTNode {
	// Pull off the next token
	tok := p.Scanner.Emit()

	if tok.Type != scanner.TOK_KEYWORD || (tok.Lexeme != "all" && tok.Lexeme != "sample") {
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected quantifier (all, sample, etc.)", tok.Lexeme)))
	}

	q := ast.QuantifierNode{
		BaseNode: ast.BaseNode{Token: tok},
		Type:     tok,
	}

	if tok.Lexeme == "sample" {
		tok = p.Scanner.Emit()
		if tok.Type != scanner.TOK_PAREN_L {
			panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected '('", tok.Lexeme)))
		}

		q.TimeQuantity = p.timeQuantity()

		tok = p.Scanner.Emit()
		if tok.Type != scanner.TOK_PAREN_R {
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
//	topic           = "/" 1*(ALPHA / DIGIT / "/")
func (p *Parser) topicSelector() ast.ASTNode {
	// Pull off the next token
	tok := p.Scanner.Emit()

	// Ensure it is the "in" keyword
	if tok.Type != scanner.TOK_KEYWORD || tok.Lexeme != "in" {
		// topic-selector is optional, so don't error out
		p.Scanner.Rewind()
		return nil
	}
	t := ast.TopicSelectorNode{In: tok.Location}

	tok = p.Scanner.Emit()
	if tok.Type != scanner.TOK_TOPIC && tok.Type != scanner.TOK_SLASH {
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected a topic after 'in' keyword", tok.Lexeme)))
	}

	t.Topic = tok

	return &t
}

// timePredicate returns a TimePredicateNode
//
// Grammar:
//
//	time-predicate  = ( "since" time-expression ) / ( "before" time-expression ) /
//	                ( "between" time-expression "," time-expression )
func (p *Parser) timePredicate() ast.ASTNode {
	tok := p.Scanner.Emit()

	if tok.Type != scanner.TOK_KEYWORD || (tok.Lexeme != "since" && tok.Lexeme != "before" &&
		tok.Lexeme != "between") {
		// time-predicates are optional, so don't error out
		p.Scanner.Rewind()
		return nil
	}

	lh := p.timeExpression()

	t := ast.TimePredicateNode{BaseNode: ast.BaseNode{
		Token: tok,
	}}
	t.Begin = lh

	if tok.Lexeme == "between" {
		comma := p.Scanner.Emit()

		if comma.Lexeme != "," {
			panic(parse.NewSyntaxError(comma, fmt.Sprintf("Error: unexpected token '%s', expected ','", comma.Lexeme)))
		}

		t.Comma = comma.Location

		rh := p.timeExpression()
		t.End = rh
	}

	return &t
}

// timeExpression returns a TimeExpressionNode
//
// Grammar:
//
//	time-expression = ( time-whence ( "-" / "+" ) time-quantity ) / time-whence
func (p *Parser) timeExpression() ast.ASTNode {
	whence := p.timeWhence()

	t := ast.TimeExpressionNode{}
	t.Whence = whence

	tok := p.Scanner.Emit()
	if tok.Lexeme == "-" || tok.Lexeme == "+" {
		t.Token = tok
		t.Op = tok
		t.Quantity = p.timeQuantity()
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
func (p *Parser) timeWhence() ast.ASTNode {
	tok := p.Scanner.Emit()

	if tok.Type != scanner.TOK_WHENCE {
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

	return &ast.TimeWhenceNode{
		BaseNode: ast.BaseNode{
			Token: tok,
		},
		When: when,
	}
}

// timeQuantity returns either the result of a single timeTerm, or a BinaryOpNode
//
// Grammar:
//
//	time-quantity   = time-term *( ( "-" / "+" ) time-term )
func (p *Parser) timeQuantity() ast.ASTNode {
	lh := p.timeTerm()

	tok := p.Scanner.Emit()
	if tok.Lexeme != "-" && tok.Lexeme != "+" {
		p.Scanner.Rewind()
		return lh
	}

	node := ast.BinaryOpNode{BaseNode: ast.BaseNode{
		Token: tok,
	}}

	node.Op = tok

	rh := p.timeTerm()

	node.Left = lh
	node.Right = rh
	return &node
}

// timeTerm returns the result of a timeAtom, or a BinaryOpNode
//
// Grammar:
//
//	time-term       = time-atom *( "*" time-atom )
func (p *Parser) timeTerm() ast.ASTNode {
	lh := p.timeAtom()

	tok := p.Scanner.Emit()
	if tok.Lexeme != "*" {
		p.Scanner.Rewind()
		return lh
	}

	node := ast.BinaryOpNode{BaseNode: ast.BaseNode{
		Token: tok,
	}}
	node.Op = tok

	rh := p.timeAtom()

	node.Left = lh
	node.Right = rh
	return &node
}

// timeAtom returns a NumberNode, or a TimespanNode
//
// Grammar:
//
//	time-atom       = number / timespan
func (p *Parser) timeAtom() ast.ASTNode {
	tok := p.Scanner.Emit()

	switch tok.Type {
	case scanner.TOK_INTEGER:
		return ast.MakeNumberNode(tok)
	case scanner.TOK_TIMESPAN:
		return &ast.TimespanNode{BaseNode: ast.BaseNode{
			Token: tok,
		}}
	}

	panic(parse.NewSyntaxError(tok, fmt.Sprintf("Expected number of timespan, got '%s'", tok.Lexeme)))
}

// dataPipeline returns a DataPipelineNode, or nil
//
// Grammar:
//
//	data-pipeline   = 1*data-stage
func (p *Parser) dataPipeline() ast.ASTNode {
	stage := p.dataStage()
	if stage == nil {
		return nil
	}

	stages := []ast.ASTNode{}

	for stage != nil {
		// Chain our stages together
		if len(stages) > 0 {
			lastStage := stages[len(stages)-1].(*ast.DataFunctionNode)
			lastStage.Next = stage.(*ast.DataFunctionNode)
		}
		stages = append(stages, stage)
		stage = p.dataStage()
	}

	return &ast.DataPipelineNode{Stages: stages}
}

// dataStage returns a DataFunctionNode, or nil if it's not a stage
//
// Grammar:
//
//	data-stage      = "|" data-function
func (p *Parser) dataStage() ast.ASTNode {
	t := p.Scanner.Emit()
	if t.Type != scanner.TOK_PIPE {
		p.Scanner.Rewind()
		return nil
	}

	return p.dataFunction()
}

// dataFunction returns a DataFunctionNode, or errors out
//
// Grammar:
//
//	data-function   = ( "filter" / "map" / "reduce" ) data-args "->" ( expression / composite / tuple )
//	data-args       = identifier [ "," data-args ]
func (p *Parser) dataFunction() ast.ASTNode {
	t := p.Scanner.Emit()
	if t.Type != scanner.TOK_KEYWORD && t.Lexeme != "map" && t.Lexeme != "reduce" &&
		t.Lexeme != "filter" {
		panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Unexpected token '%s', expected 'filter', 'map', or 'reduce'", t.Lexeme)))
	}

	fn := ast.DataFunctionNode{BaseNode: ast.BaseNode{Token: t}, Name: t}

	// First, parse arguments
	t = p.Scanner.Emit()
	for {
		// We're done parsing arguments when we hit a '->'
		if t.Type == scanner.TOK_ARROW {
			break
		}

		if t.Type != scanner.TOK_IDENTIFIER {
			panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Unexpected token '%s', expected an identifier", t.Lexeme)))
		}

		fn.Arguments = append(fn.Arguments, ast.IdentifierNode{BaseNode: ast.BaseNode{Token: t}})

		// Pull off a comma if one exists
		t = p.Scanner.Emit()

		if t.Type != scanner.TOK_COMMA {
			continue
		}

		// Next argument
		t = p.Scanner.Emit()
	}

	fn.Expression = p.composite()

	if fn.Expression == nil {
		fn.Expression = p.tuple()
	}

	return &fn
}

// expression returns a BinaryOpNode, or the result of comparison
//
// Grammar:
//
//	expression      = comparison *( ( "!=" / "==" ) expression )
func (p *Parser) expression() ast.ASTNode {
	c := p.comparison()

	t := p.Scanner.Emit()
	if t.Type == scanner.TOK_NOT_EQ || t.Type == scanner.TOK_EQ_EQ {
		op := ast.BinaryOpNode{BaseNode: ast.BaseNode{Token: t}}
		op.Op = t
		op.Left = c
		op.Right = p.expression()
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
func (p *Parser) comparison() ast.ASTNode {
	t := p.term()

	c := p.Scanner.Emit()
	if c.Type == scanner.TOK_GREATER || c.Type == scanner.TOK_GREATER_EQ ||
		c.Type == scanner.TOK_LESS || c.Type == scanner.TOK_LESS_EQ {
		op := ast.BinaryOpNode{BaseNode: ast.BaseNode{Token: c}}
		op.Op = c
		op.Left = t
		op.Right = p.comparison()
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
func (p *Parser) term() ast.ASTNode {
	t := p.termMD()

	c := p.Scanner.Emit()
	if c.Type == scanner.TOK_MINUS || c.Type == scanner.TOK_PLUS {
		op := ast.BinaryOpNode{BaseNode: ast.BaseNode{Token: c}}
		op.Op = c
		op.Left = t
		op.Right = p.term()
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
func (p *Parser) termMD() ast.ASTNode {
	u := p.unary()

	c := p.Scanner.Emit()
	if c.Type == scanner.TOK_SLASH || c.Type == scanner.TOK_STAR {
		op := ast.BinaryOpNode{BaseNode: ast.BaseNode{Token: c}}
		op.Op = c
		op.Left = u
		op.Right = p.termMD()
		return &op
	}
	p.Scanner.Rewind()

	return u
}

// unary returns a UnaryOpNode, or the result of primary
//
// Grammar:
//
//	unary           = ( ( "-" / "+" ) ( sub-value / integer / float / identifier ) ) / composite / primary
func (p *Parser) unary() ast.ASTNode {
	t := p.Scanner.Emit()
	if t.Type == scanner.TOK_MINUS || t.Type == scanner.TOK_PLUS {
		op := ast.UnaryOpNode{BaseNode: ast.BaseNode{Token: t}, Operator: t}

		tupleVal := p.subValue()
		if tupleVal != nil {
			op.Operand = tupleVal
			return &op
		}

		t = p.Scanner.Emit()

		if t.Type == scanner.TOK_INTEGER || t.Type == scanner.TOK_FLOAT {
			op.Operand = ast.MakeNumberNode(t)
		} else if t.Type == scanner.TOK_IDENTIFIER {
			op.Operand = &ast.IdentifierNode{ast.BaseNode{Token: t}}
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
//	primary         = builtin / sub-value / identifier / integer / float / string / "(" expression ")"
func (p *Parser) primary() ast.ASTNode {
	builtin := p.builtin()
	if builtin != nil {
		return builtin
	}

	tupleVal := p.subValue()
	if tupleVal != nil {
		return tupleVal
	}

	t := p.Scanner.Emit()

	switch t.Type {
	case scanner.TOK_INTEGER, scanner.TOK_FLOAT:
		return ast.MakeNumberNode(t)
	case scanner.TOK_STRING:
		return ast.MakeStringNode(t)
	case scanner.TOK_PAREN_L:
		// We're an expression group, so call expression
		expr := p.expression()

		// Expect a closing paren
		t = p.Scanner.Emit()
		if t.Type != scanner.TOK_PAREN_R {
			panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Unexpected token '%s'. Expected a ')'", t.Lexeme)))
		}

		return expr
	default:
		panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Unexpected token '%s'. Expected identifier, number, string, tuple or builtin.", t.Lexeme)))
	}
}

// subValue returns a ElementNode, or Identifier if there is no subscript.
//
// Grammar:
//
//	sub-value     = identifier "[" ( integer / string ) "]"
func (p *Parser) subValue() ast.ASTNode {
	t := p.Scanner.Emit()

	if t.Type != scanner.TOK_IDENTIFIER {
		p.Scanner.Rewind()
		return nil
	}

	identifier := ast.IdentifierNode{BaseNode: ast.BaseNode{Token: t}}

	t = p.Scanner.Emit()

	// If we're not a tuple value, return the identifier
	if t.Type != scanner.TOK_BRACKET_L {
		p.Scanner.Rewind()
		return &identifier
	}

	t = p.Scanner.Emit()

	var subscript ast.ASTNode

	switch t.Type {
	case scanner.TOK_INTEGER:
		subscript = ast.MakeNumberNode(t)
	case scanner.TOK_STRING:
		subscript = ast.MakeStringNode(t)
	case scanner.TOK_IDENTIFIER:
		subscript = ast.MakeStringNodeFromID(t)
	default:
		panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Invalid tuple subscript '%s', expected an integer or string.", t.Lexeme)))
	}

	t = p.Scanner.Emit()

	if t.Type != scanner.TOK_BRACKET_R {
		panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Expected ']'")))
	}

	return &ast.ElementNode{Identifier: identifier, Subscript: subscript}
}

// builtin returns a BuiltinFunctionNode or an IdentifierNode or nil
//
// Grammar:
//
//	builtin         = identifier "(" expression  ")"
func (p *Parser) builtin() ast.ASTNode {
	start, pos := p.Scanner.Start, p.Scanner.Pos
	t := p.Scanner.Emit()

	if t.Type != scanner.TOK_IDENTIFIER {
		p.Scanner.Rewind()
		return nil
	}

	node := ast.BuiltinFunctionNode{BaseNode: ast.BaseNode{Token: t}, Name: t}

	t = p.Scanner.Emit()
	if t.Type != scanner.TOK_PAREN_L {
		p.Scanner.Start, p.Scanner.Pos = start, pos
		return nil
	}
	node.LParen = t.Location

	node.Expression = p.tuple()

	t = p.Scanner.Emit()
	if t.Type != scanner.TOK_PAREN_R {
		panic(parse.NewSyntaxError(t, fmt.Sprintf("Error: Unexpected token '%s'. Expected ')'", t.Lexeme)))
	}
	node.RParen = t.Location

	return &node
}

// tuple returns a TupleNode or the result of expression
//
// Grammar:
//
//	tuple           = expression 1*( "," expression )
func (p *Parser) tuple() ast.ASTNode {
	list := ast.TupleNode{}
	e := p.expression()
	t := p.Scanner.Emit()
	list.Elements = append(list.Elements, e)

	for t.Type == scanner.TOK_COMMA {
		e = p.expression()
		list.Elements = append(list.Elements, e)
		t = p.Scanner.Emit()
	}

	p.Scanner.Rewind()

	// We weren't a tuple, so restore scanner, and return nil
	if len(list.Elements) == 1 {
		return e
	}

	return &list
}

// composite returns a CompositeNode
//
// Grammar:
//
//	composite      = string ":" primary *( "," string ":" primary )
func (p *Parser) composite() ast.ASTNode {
	composite := ast.CompositeNode{}
	found := false

	for {
		start, pos := p.Scanner.Start, p.Scanner.Pos
		t := p.Scanner.Emit()

		var key *ast.StringNode

		switch t.Type {
		case scanner.TOK_STRING:
			key = ast.MakeStringNode(t)
		case scanner.TOK_IDENTIFIER:
			key = ast.MakeStringNodeFromID(t)
		default:
			p.Scanner.Rewind()
			break
		}

		t = p.Scanner.Emit()

		if t.Type != scanner.TOK_COLON {
			p.Scanner.Start, p.Scanner.Pos = start, pos
			break
		}

		value := p.expression()

		composite.Keys = append(composite.Keys, *key)
		composite.Values = append(composite.Values, value)
		found = true

		// If next token is not a comma, we're done
		t = p.Scanner.Emit()

		if t.Type != scanner.TOK_COMMA {
			p.Scanner.Rewind()
			break
		}
	}

	if found == false {
		return nil
	}

	return &composite
}
