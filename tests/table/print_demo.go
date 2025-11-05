package main

import (
	"bytes"
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	homerun "github.com/stuttgart-things/homerun-library" // use module path from go.mod
)

func main() {
	// Define table
	header := table.Row{"Name", "Age"}
	row := table.Row{"Charlie", 28}
	style := table.StyleLight

	var buf bytes.Buffer

	// Call your library function
	homerun.PrintTable(&buf, header, row, style)

	// Print the result
	fmt.Println(buf.String())
}
