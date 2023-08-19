/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package schema

import (
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestType_Validate(t *testing.T) {
	tt := Type{Name: "int8"}

	if !tt.Validate([]byte{byte(12)}) {
		t.Fail()
	}

	tt = Type{Name: "int16"}

	b := make([]byte, 2)
	binary.LittleEndian.AppendUint16(b, 12)
	if !tt.Validate(b) {
		t.Fail()
	}
}

func TestArray_Validate(t *testing.T) {
	tt := Array{Type: Type{Name: "int32"}, Length: 10}

	b := make([]byte, 40)
	if !tt.Validate(b) {
		t.Fail()
	}
}

func TestJSONMarshal(t *testing.T) {
	ta := Array{Type: Type{Name: "int32"}, Length: 10}

	b, _ := json.Marshal(ta)
	if string(b) != `"[10]int32"` {
		t.Fail()
	}
}
