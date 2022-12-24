/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import (
	"errors"
	"fmt"
	"github.com/dburkart/fossil/pkg/common/parse"
	"strings"
)

type Parser struct {
	Scanner Scanner
}

func (p *Parser) Parse() (schema Object, err error) {
	defer func() {
		if e := recover(); e != nil {
			syntaxError, ok := e.(parse.SyntaxError)
			if !ok {
				panic(e)
			}
			err = errors.New(syntaxError.FormatError(p.Scanner.Input))
		}
	}()

	p.Scanner.Input = strings.Trim(p.Scanner.Input, " \t\n")

	schema = p.schema()

	return
}

func (p *Parser) schema() Object {
	var schema Object

	if schema = p.dType(); schema != nil {
		return schema
	}

	if schema = p.array(); schema != nil {

	}

	return schema
}

func (p *Parser) dType() Object {
	var dType *Type

	tok := p.Scanner.Emit()
	if tok.Type != TOK_TYPE {
		p.Scanner.Rewind()
		return nil
	}

	dType.Name = tok.Lexeme

	return dType
}

func (p *Parser) array() Object {
	var array *Array

	tok := p.Scanner.Emit()
	if tok.Type != TOK_BRACKET_O {
		p.Scanner.Rewind()
		return nil
	}

	tok = p.Scanner.Emit()
	if tok.Type != TOK_NUMBER {
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected a number indicating array size", tok.Lexeme)))
	}

	return array
}
