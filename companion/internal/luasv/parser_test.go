package luasv

import (
	"strings"
	"testing"
)

func TestParseVariable(t *testing.T) {
	input := `
OTHER = { ["ignored"] = true }
WOW_MARKETS_DB = {
  ["schemaVersion"] = 1,
  ["config"] = {
    ["maxExportRows"] = 100,
  },
  ["values"] = {
    [1] = "first",
    [2] = "line\nsecond",
    [3] = 42,
  },
}
`

	root, err := ParseVariable(strings.NewReader(input), "WOW_MARKETS_DB")
	if err != nil {
		t.Fatalf("ParseVariable() error = %v", err)
	}

	if value, ok := root.Field("schemaVersion"); !ok || value != int64(1) {
		t.Fatalf("schemaVersion = %#v, %v", value, ok)
	}

	rawValues, ok := root.Field("values")
	if !ok {
		t.Fatal("values field not found")
	}
	values, err := rawValues.(*Table).Sequence()
	if err != nil {
		t.Fatalf("Sequence() error = %v", err)
	}
	if len(values) != 3 || values[1] != "line\nsecond" || values[2] != int64(42) {
		t.Fatalf("values = %#v", values)
	}
}

func TestParseVariableRejectsCode(t *testing.T) {
	input := `WOW_MARKETS_DB = os.execute("echo unsafe")`

	_, err := ParseVariable(strings.NewReader(input), "WOW_MARKETS_DB")
	if err == nil {
		t.Fatal("ParseVariable() accepted executable Lua")
	}
}

func TestSequenceRejectsGaps(t *testing.T) {
	table := &Table{
		Fields:  map[string]Value{},
		Indexed: map[int]Value{1: "first", 3: "third"},
	}

	if _, err := table.Sequence(); err == nil {
		t.Fatal("Sequence() accepted a sparse table")
	}
}

func TestImplicitSequenceUsesArrayStorage(t *testing.T) {
	root, err := ParseVariable(
		strings.NewReader(`DATA = { "first", "second", "third" }`),
		"DATA",
	)
	if err != nil {
		t.Fatalf("ParseVariable() error = %v", err)
	}
	if len(root.Array) != 3 {
		t.Fatalf("len(Array) = %d, want 3", len(root.Array))
	}
	if len(root.Indexed) != 0 {
		t.Fatalf("len(Indexed) = %d, want 0", len(root.Indexed))
	}
}
