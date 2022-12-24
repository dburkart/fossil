/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import "testing"

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
