// print_table_test.go
package homerun

import (
	"bytes"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
)

func TestPrintTable(t *testing.T) {
	// Define the header and row for the table
	header := table.Row{"Name", "Age"}
	row := table.Row{"Alice", 30}

	// Choose a table style
	style := table.StyleLight

	// Capture the output
	var buf bytes.Buffer

	// Call the function with the buffer as the output
	PrintTable(&buf, header, row, style)

	// Expected output (modify as needed based on the style chosen)
	expected := `┌───────┬─────┐
│ NAME  │ AGE │
├───────┼─────┤
│ Alice │  30 │
└───────┴─────┘
`

	// Compare the captured output to the expected output
	if buf.String() != expected {
		t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", buf.String(), expected)
	}
}
