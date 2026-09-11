/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
)

// PrintTableRows writes a table with any number of rows to output.
func PrintTableRows(output io.Writer, header table.Row, rows []table.Row, style table.Style) {
	t := table.NewWriter()
	t.SetOutputMirror(output)
	t.AppendHeader(header)
	t.AppendRows(rows)
	t.SetStyle(style)
	// Render also returns the rendered string; the table is written through the
	// output mirror set above, so it is deliberately discarded here.
	_ = t.Render()
}

// PrintTable writes a table with a single row to output.
//
// Deprecated: a table of one row is rarely what a caller needs. Use
// PrintTableRows, which takes any number of rows. PrintTable is kept so v4
// callers keep compiling and will be removed in v5 (#57).
func PrintTable(output io.Writer, header, row table.Row, style table.Style) {
	PrintTableRows(output, header, []table.Row{row}, style)
}
