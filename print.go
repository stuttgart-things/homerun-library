// homerun.go
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
	t.Render()
}
