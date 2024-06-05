package main

import (
	"net"
)

type Pokemon struct {
	Name      string `json:"name"`
	PokedexID int    `json:"id"`
}

type PlayerPokemon struct {
	Name  string
	ID    string
	Level int
	Exp   int
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
	Pokemons []PlayerPokemon
	Battle   *Battle
}

type Battle struct {
	Player1 string
	Player2 string
	Turn    string
}

var pokedex Pokedex

var players = make(map[string]*Player)

var battles = make(map[string]*Battle)
var p1pokemons = make(map[string]*PlayerPokemon)
var p2pokemons = make(map[string]*PlayerPokemon)
