package main

import (
	"fmt"
	"net"
)

// "fmt"
// "net"
// "strings"

const (
	HOST        = "localhost"
	PORT        = "8080"
	TYPE        = "udp"
	pokedexData = "JSON\\pokedex.json"
)

type Pokemon struct {
	Name string
}

type Type struct {
	Name   string   `json:"name"`
	Effect []string `json:"effectiveAgainst"`
	Weak   []string `json:"weakAgainst"`
}

type Pokedex struct {
	Types    []Type    `json:"types"`
	Pokemons []Pokemon `json:"pokemons"`
}

func main() {
	udpAddr, err := net.ResolveUDPAddr(TYPE, HOST+":"+PORT)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Game server server has been running on", udpAddr)
}
