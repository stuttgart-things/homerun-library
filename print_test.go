/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"bytes"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
)

func TestPrintTableRows(t *testing.T) {
	cases := []struct {
		name     string
		rows     []table.Row
		expected string
	}{
		{
			name: "several rows",
			rows: []table.Row{{"Alice", 30}, {"Bob", 4}, {"Charlie", 28}},
			expected: `┌─────────┬─────┐
│ NAME    │ AGE │
├─────────┼─────┤
│ Alice   │  30 │
│ Bob     │   4 │
│ Charlie │  28 │
└─────────┴─────┘
`,
		},
		{
			name: "no rows",
			rows: nil,
			expected: `┌──────┬─────┐
│ NAME │ AGE │
├──────┼─────┤
└──────┴─────┘
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			PrintTableRows(&buf, table.Row{"Name", "Age"}, tc.rows, table.StyleLight)

			if buf.String() != tc.expected {
				t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", buf.String(), tc.expected)
			}
		})
	}
}

func TestPrintTable(t *testing.T) {
	header := table.Row{"Name", "Age"}
	row := table.Row{"Alice", 30}

	var buf bytes.Buffer
	PrintTable(&buf, header, row, table.StyleLight)

	expected := `┌───────┬─────┐
│ NAME  │ AGE │
├───────┼─────┤
│ Alice │  30 │
└───────┴─────┘
`

	if buf.String() != expected {
		t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", buf.String(), expected)
	}
}
