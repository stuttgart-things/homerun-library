package main

import (
	"bytes"
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	homerun "github.com/stuttgart-things/homerun-library/v4" // use module path from go.mod
)

func main() {
	// Define table
	header := table.Row{"Name", "Age"}
	rows := []table.Row{
		{"Alice", 30},
		{"Bob", 4},
		{"Charlie", 28},
	}
	style := table.StyleLight

	var buf bytes.Buffer

	// Call your library function
	homerun.PrintTableRows(&buf, header, rows, style)

	// Print the result
	fmt.Println(buf.String())
}
