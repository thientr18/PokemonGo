package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
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

type Player struct {
	Name     string
	Addr     *net.UDPAddr
	Pokemons []Pokemon `json:"pokemons"`
}

var players = make(map[string]*Player)

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

func handlePokedex() {
	file, err := os.Open(pokedexData)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	jsonData, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Error:", err)
		return
	}

	var pokemons []Pokemon
	err = json.Unmarshal(jsonData, &pokemons)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}

func handleLevel() {

}
