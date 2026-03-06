// print_table_test.go
package homerun

import (
	"bytes"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
)

func TestPrintTable(t *testing.T) {
	header := table.Row{"Name", "Age"}
	rows := []table.Row{
		{"Alice", 30},
		{"Bob", 25},
	}

	style := table.StyleLight

	var buf bytes.Buffer

	PrintTable(&buf, header, rows, style)

	expected := `┌───────┬─────┐
│ NAME  │ AGE │
├───────┼─────┤
│ Alice │  30 │
│ Bob   │  25 │
└───────┴─────┘
`

	if buf.String() != expected {
		t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", buf.String(), expected)
	}
}
