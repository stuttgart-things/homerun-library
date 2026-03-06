package main

import (
	"bytes"
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	homerun "github.com/stuttgart-things/homerun-library" // use module path from go.mod
)

func main() {
	header := table.Row{"Name", "Age"}
	rows := []table.Row{
		{"Charlie", 28},
		{"Diana", 32},
	}
	style := table.StyleLight

	var buf bytes.Buffer

	homerun.PrintTable(&buf, header, rows, style)

	fmt.Println(buf.String())
}
