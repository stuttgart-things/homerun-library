package homerun

import (
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
)

// PrintTable writes a table to the specified output writer
func PrintTable(output io.Writer, header table.Row, rows []table.Row, style table.Style) {
	t := table.NewWriter()
	t.SetOutputMirror(output)
	t.AppendHeader(header)
	t.AppendRows(rows)
	t.SetStyle(style)
	t.Render()
}
