/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import (
	"strings"
	"testing"
)

func TestParseType(t *testing.T) {
	p := Parser{
		Scanner{
			Input: "int32",
		},
	}

	_, err := p.Parse()
	if err != nil {
		t.Fail()
	}

	p = Parser{
		Scanner{
			Input: "uint32",
		},
	}

	_, err = p.Parse()
	if err != nil {
		t.Fail()
	}

	p = Parser{
		Scanner{
			Input: "string",
		},
	}

	_, err = p.Parse()
	if err != nil {
		t.Fail()
	}

	p = Parser{
		Scanner{
			Input: "bogus",
		},
	}

	_, err = p.Parse()
	if err == nil {
		t.Fail()
	}

	p = Parser{
		Scanner{
			Input: "float32",
		},
	}

	_, err = p.Parse()
	if err != nil {
		t.Fail()
	}
}

func TestParseArray(t *testing.T) {
	p := Parser{
		Scanner{
			Input: "[2]int32",
		},
	}

	_, err := p.Parse()
	if err != nil {
		t.Fail()
	}

	p = Parser{
		Scanner{
			Input: "[]int32",
		},
	}

	_, err = p.Parse()
	if err == nil {
		t.Fail()
	}

	p = Parser{
		Scanner{
			Input: "[foo]int32",
		},
	}

	_, err = p.Parse()
	if err == nil {
		t.Fail()
	}

	p = Parser{
		Scanner{
			Input: "[2]string",
		},
	}

	_, err = p.Parse()
	if err == nil {
		t.Fail()
	}
}

func slicesEqualStr(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if strings.Compare(v, b[i]) != 0 {
			return false
		}
	}
	return true
}

func slicesEqualObj(a, b []Object) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if strings.Compare(v.ToSchema(), b[i].ToSchema()) != 0 {
			return false
		}
	}
	return true
}

func TestParseShallowMap(t *testing.T) {
	p := Parser{
		Scanner{
			Input: `{ "x": int32, "y": int32, }`,
		},
	}

	obj, err := p.Parse()
	if err != nil {
		t.Fail()
	}

	if _, ok := obj.(*Composite); !ok {
		t.Fail()
	}

	p = Parser{
		Scanner{
			Input: `{ "event": string, "coords": [2]int32, }`,
		},
	}

	obj, err = p.Parse()
	if err != nil {
		t.Fail()
	}

	if _, ok := obj.(*Composite); !ok {
		t.Fail()
	}

	if !slicesEqualStr(obj.(*Composite).Keys, []string{"coords", "event"}) {
		t.Errorf("%v", obj.(*Composite).Keys)
	}

	if !slicesEqualObj(obj.(*Composite).Values, []Object{Array{2, Type{"int32"}}, Type{"string"}}) {
		t.Errorf("%v", obj.(*Composite).Keys)
	}

	p = Parser{
		Scanner{
			Input: `{ "key": foo, }`,
		},
	}

	_, err = p.Parse()
	if err == nil {
		t.Fail()
	}
}
