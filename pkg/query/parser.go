/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package query

import "fmt"

type Parser struct {
	Scanner Scanner
}

func (p *Parser) Parse() ASTNode {
	return p.query()
}

func (p *Parser) query() ASTNode {
	q := QueryNode{BaseNode{
		Value: p.Scanner.Input,
	}}

	// Queries must start with a Quantifier
	q.AddChild(p.quantifier())

	return &q
}

func (p *Parser) quantifier() ASTNode {
	// Pull off the next token
	tok := p.Scanner.Emit()

	if tok.Type != TOK_KEYWORD || tok.Lexeme != "all" {
		panic(fmt.Sprintf("Found '%s', expected quantifier (all, sample, etc.)", tok.Lexeme))
	}

	q := QuantifierNode{BaseNode{
		Value: tok.Lexeme,
	}}

	return &q
}
