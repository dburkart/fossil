package schema

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestEncodeStringForSchemaCompositeWithArray(t *testing.T) {
	composite := &Composite{
		Keys: []string{"coords", "type"},
		Values: []Object{
			&Array{Length: 2, Type: Type{Name: "int32"}},
			&Type{Name: "string"},
		},
	}

	data, err := EncodeStringForSchema("type: click, coords: 1, 2", composite)
	if err != nil {
		t.Fatalf("expected composite literal with embedded array to be parsed without error, got %v", err)
	}

	expected := make([]byte, 0, 17)
	expected = binary.LittleEndian.AppendUint32(expected, uint32(1))
	expected = binary.LittleEndian.AppendUint32(expected, uint32(2))
	expected = binary.LittleEndian.AppendUint32(expected, uint32(len("click")))
	expected = append(expected, []byte("click")...)

	if !bytes.Equal(expected, data) {
		t.Fatalf("unexpected encoding produced. expected %v, got %v", expected, data)
	}
}

func TestEncodeStringForSchemaCompositeQuotedString(t *testing.T) {
	composite := &Composite{
		Keys: []string{"coords", "message"},
		Values: []Object{
			&Array{Length: 2, Type: Type{Name: "int32"}},
			&Type{Name: "string"},
		},
	}

	_, err := EncodeStringForSchema("coords: 10, 20, message: \"hello, world\"", composite)
	if err != nil {
		t.Fatalf("expected quoted string containing comma to be parsed without error, got %v", err)
	}
}

func TestEncodeStringForSchemaCompositeTrailingComma(t *testing.T) {
	composite := &Composite{
		Keys: []string{"coords", "type"},
		Values: []Object{
			&Array{Length: 2, Type: Type{Name: "int32"}},
			&Type{Name: "string"},
		},
	}

	if _, err := EncodeStringForSchema("type: click, coords: 1, 2,", composite); err == nil {
		t.Fatalf("expected trailing comma to be reported as malformed composite literal")
	}
}
