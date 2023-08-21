/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/dburkart/fossil/pkg/common/parse"
)

func Parse(s string) (Object, error) {
	p := Parser{Scanner{Input: s}}
	return p.Parse()
}

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
	if schema == nil {
		syntaxError := parse.NewSyntaxError(parse.Token{
			Type:     TOK_INVALID,
			Location: parse.Location{Start: p.Scanner.Pos, End: len(p.Scanner.Input) - 1},
		}, "Error: Unrecognized schema")
		err = errors.New(syntaxError.FormatError(p.Scanner.Input))
	} else if p.Scanner.Pos != len(p.Scanner.Input) {
		syntaxError := parse.NewSyntaxError(parse.Token{
			Type:     TOK_INVALID,
			Location: parse.Location{Start: p.Scanner.Pos, End: len(p.Scanner.Input) - 1},
		}, "Error: Schema not valid, starting here")
		err = errors.New(syntaxError.FormatError(p.Scanner.Input))
	}

	return
}

func (p *Parser) schema() Object {
	var schema Object

	if schema = p.dType(); schema != nil {
		return schema
	}

	if schema = p.array(); schema != nil {
		return schema
	}

	if schema = p.composite(); schema != nil {
		return schema
	}

	return nil
}

func (p *Parser) dType() Object {
	var dType Type

	tok := p.Scanner.Emit()
	if tok.Type != TOK_TYPE {
		p.Scanner.Rewind()
		return nil
	}

	dType.Name = tok.Lexeme

	return &dType
}

func (p *Parser) array() Object {
	var array Array
	var err error

	tok := p.Scanner.Emit()
	if tok.Type != TOK_BRACKET_O {
		p.Scanner.Rewind()
		return nil
	}

	tok = p.Scanner.Emit()
	if tok.Type != TOK_NUMBER {
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected a number indicating array size", tok.Lexeme)))
	}

	array.Length, err = strconv.Atoi(tok.Lexeme)
	if err != nil {
		panic(err)
	}

	tok = p.Scanner.Emit()
	if tok.Type != TOK_BRACKET_X {
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected a ']'", tok.Lexeme)))
	}

	obj := p.dType()
	if obj == nil {
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected a valid type", tok.Lexeme)))
	}

	dType := obj.(*Type)
	if dType == nil {
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected a valid type", tok.Lexeme)))
	}

	if dType.Name == "string" || dType.Name == "binary" {
		panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: variable-length type '%s' not valid in array", dType.Name)))
	}

	array.Type = *dType

	return &array
}

// Insert a string into a list of strings (preserving order), returning the resulting index
func insertInto(list []string, key string) ([]string, int) {
	for idx, s := range list {
		if strings.Compare(key, s) <= 0 {
			return append(list[:idx], append([]string{key}, list[idx:]...)...), idx
		}
	}
	return append(list, key), len(list)
}

func (p *Parser) composite() Object {
	var composite Composite

	tok := p.Scanner.Emit()
	if tok.Type != TOK_CURLY_O {
		p.Scanner.Rewind()
		return nil
	}

	tok = p.Scanner.Emit()

	// Iterate over the keys and values of our map
	for tok.Type != TOK_CURLY_X {
		if tok.Type != TOK_KEY {
			panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected a map key (\"...\")", tok.Lexeme)))
		}

		var idx int
		unquotedKey, err := strconv.Unquote(tok.Lexeme)
		if err != nil {
			unquotedKey = tok.Lexeme
		}
		composite.Keys, idx = insertInto(composite.Keys, unquotedKey)

		tok = p.Scanner.Emit()
		if tok.Type != TOK_COLON {
			panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected ':'", tok.Lexeme)))
		}

		// Now we must find a valid type
		val := p.dType()
		if val == nil {
			// It could be an array
			val = p.array()

			if val == nil {
				panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: expected a valid type or array as value for key '%s'", tok.Lexeme)))
			}
		}

		composite.Values = append(composite.Values[:idx], append([]Object{val}, composite.Values[idx:]...)...)

		// Finally, every line must have a comma
		tok = p.Scanner.Emit()

		// Our composite could be over now
		if tok.Type == TOK_CURLY_X {
			break
		}

		if tok.Type != TOK_COMMA {
			panic(parse.NewSyntaxError(tok, fmt.Sprintf("Error: unexpected token '%s', expected ','", tok.Lexeme)))
		}

		// Pull off token for the next iteration
		tok = p.Scanner.Emit()
	}

	return &composite
}
