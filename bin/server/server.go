package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
)

type (
	Pokemon struct {
		Id       string   `json:"ID"`
		Name     string   `json:"Name"`
		Types    []string `json:"types"`
		Link     string   `json:"URL"`
		PokeInfo PokeInfo `json:"Poke-Information"`
	}

	PokeInfo struct {
		Hp    int `json:"HP"`
		Atk   int `json:"ATK"`
		Def   int `json:"DEF"`
		SpAtk int `json:"Sp.Atk"`
		SpDef int `json:"Sp.Def"`
		Speed int `json:"Speed"`
	}

	PlayerPokemon struct { // store pokemmon that a player holding
		Name  string `json:"Name"`
		ID    string
		Level int
		Exp   int
		Hp    int `json:"HP"`
		Atk   int `json:"ATK"`
		Def   int `json:"DEF"`
		SpAtk int `json:"Sp.Atk"`
		SpDef int `json:"Sp.Def"`
		Speed int `json:"Speed"`
	}

	Player struct {
		Name                  string
		Pokemons              map[string]PlayerPokemon
		battleRequestSends    map[string]string // store number of request that a player send: 'map[receivers]sender'
		battleRequestReceives map[string]string // store number of request that a player get: 'map[senders]receiver'
		Active                string
	}

	BattlePokemon struct { // 3 pokemons to choose in a battle
		Name  string `json:"Name"`
		Hp    int    `json:"HP"`
		Atk   int    `json:"ATK"`
		Def   int    `json:"DEF"`
		SpAtk int    `json:"Sp.Atk"`
		SpDef int    `json:"Sp.Def"`
		Speed int    `json:"Speed"`
	}

	Game struct {
		TurnOrder []string
		Current   int
		Players   map[string]*Player
		Status    string // "waiting", "inviting", "active"
	}

	GameState struct {
		mu    sync.Mutex
		Games map[string]*Game
	}
)

var gameState = GameState{
	Games: make(map[string]*Game),
}

var availablePokemons []PlayerPokemon

func loadPokemonData(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var pokemons struct {
		Pokemons []PlayerPokemon `json:"pokemons"`
	}
	if err := json.Unmarshal(data, &pokemons); err != nil {
		return err
	}
	availablePokemons = pokemons.Pokemons
	return nil
}

func formatPokemonList() string {
	var sb strings.Builder
	for _, p := range availablePokemons {
		sb.WriteString(fmt.Sprintf("%s (HP: %d, Attack: %d)\n", p.Name, p.Hp, p.Atk))
	}
	return sb.String()
}

func main() {
	if err := loadPokemonData("test\\pokemon_data.json"); err != nil {
		panic(err)
	}
	fmt.Println("USE FOR LOOP: ")
	for _, p := range availablePokemons {
		hehe := fmt.Sprintf("%s (HP: %d, Attack: %d)\n", p.Name, p.Hp, p.Atk)
		fmt.Printf(hehe)
	}

	inBattle["hehe"] = false
	fmt.Println(checkInBattle("hehe"))
}

var inBattle = make(map[string]bool)

func checkInBattle(s string) bool {
	_, exists := inBattle[s]
	if !exists {
		return false
	} else {
		return true
	}
}
