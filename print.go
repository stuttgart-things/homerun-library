/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
)

// PrintTable writes a table to the specified output writer
func PrintTable(output io.Writer, header, row table.Row, style table.Style) {
	t := table.NewWriter()
	t.SetOutputMirror(output)
	t.AppendHeader(header)
	t.AppendRow(row)
	t.SetStyle(style)
	// Render also returns the rendered string; the table is written through the
	// output mirror set above, so it is deliberately discarded here. Returning
	// it would change the signature - see #57.
	_ = t.Render()
}
