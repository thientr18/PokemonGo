package main

import (
// "fmt"
// "net"
// "strings"
)

type Pokemon struct {
	Name string
}

type Pokedex struct {
	Types []Type `json:"types"`
}

type Type struct {
	Name   string   `json:"name"`
	Effect []string `json:"effectiveAgainst"`
	Weak   []string `json:"weakAgainst"`
}

func main() {

}
