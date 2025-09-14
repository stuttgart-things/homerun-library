package main

import (
	"fmt"

	"github.com/stuttgart-things/homerun-library" // use module path from go.mod
)

func main() {
	items := []string{"apple", "banana", "cherry", "date"}
	fmt.Println("Random item selected:", homerun.GetRandomObject(items))
}
