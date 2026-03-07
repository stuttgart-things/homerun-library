package main

import (
	"fmt"

	homerun "github.com/stuttgart-things/homerun-library/v2" // use module path from go.mod
)

func main() {
	items := []string{"apple", "banana", "cherry", "date"}
	fmt.Println("Random item selected:", homerun.GetRandomObject(items))
}
