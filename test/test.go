package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

const (
	HOST               = "localhost"
	PORT               = "8080"
	TYPE               = "udp"
	pokedexData        = "src\\pokedex.json"
	playerpokemonsData = "src\\playersPokemon.json"
)

type (
	Pokemon struct {
		Id       string   `json:"ID"`
		Name     string   `json:"Name"`
		Link     string   `json:"URL"`
		PokeInfo PokeInfo `json:"Poke-Information"`
	}

	PokeInfo struct {
		Types       []string `json:"types"`
		Hp          int      `json:"HP"`
		Atk         int      `json:"ATK"`
		Def         int      `json:"DEF"`
		SpAtk       int      `json:"Sp.Atk"`
		SpDef       int      `json:"Sp.Def"`
		Speed       int      `json:"Speed"`
		TypeDefense TypeDef  `json:"Type-Defenses"`
	}
	TypeDef struct {
		Normal   float32
		Fire     float32
		Water    float32
		Electric float32
		Grass    float32
		Ice      float32
		Fighting float32
		Poison   float32
		Ground   float32
		Flying   float32
		Psychic  float32
		Bug      float32
		Rock     float32
		Ghost    float32
		Dragon   float32
		Dark     float32
		Steel    float32
		Fairy    float32
	}

	PlayerPokemon struct { // store pokemmon that a player holding
		Owner          string           `json:"PlayerName"`
		PlayerPokeInfo []PlayerPokeInfo `json:"Pokemons"`
	}
	PlayerPokeInfo struct { // store pokemmon that a player holding
		Name        string   `json:"Name"`
		ID          string   `json:"ID"`
		Level       int      `json:"Level"`
		Exp         int      `json:"Exp"`
		Types       []string `json:"types"`
		Hp          int      `json:"HP"`
		Atk         int      `json:"ATK"`
		Def         int      `json:"DEF"`
		SpAtk       int      `json:"Sp.Atk"`
		SpDef       int      `json:"Sp.Def"`
		Speed       int      `json:"Speed"`
		TypeDefense TypeDef  `json:"Type-Defenses"`
	}
	Player struct {
		Name                  string
		Pokemons              map[string]PlayerPokemon // string là pokemon ID
		BattlePokemon         map[string]BattlePokemon
		battleRequestSends    map[string]string // store number of request that a player send: 'map[receivers]sender'
		battleRequestReceives map[string]string // store number of request that a player get: 'map[senders]receiver'
		Active                string
		battleID              int64
	}

	BattlePokemon struct {
		battleID    int64
		Owner       string   `json:"PlayerName"`
		Name        string   `json:"Name"`
		ID          string   `json:"ID"`
		Level       int      `json:"Level"`
		Exp         int      `json:"Exp"`
		Types       []string `json:"types"`
		Hp          int      `json:"HP"`
		Atk         int      `json:"ATK"`
		Def         int      `json:"DEF"`
		SpAtk       int      `json:"Sp.Atk"`
		SpDef       int      `json:"Sp.Def"`
		Speed       int      `json:"Speed"`
		TypeDefense TypeDef  `json:"Type-Defenses"`
	}
)

var pokedex []Pokemon               // pokedex
var playersPokemons []PlayerPokemon // player's Pokemons

func main() {
	// Load the pokedex data from the JSON file
	err := loadPokedex(pokedexData)
	if err != nil {
		fmt.Println("Error loading pokedex data:", err)
	} else {
		fmt.Println("Load thành công")
	}

	err = loadPlayerPokemons(playerpokemonsData)
	if err != nil {
		fmt.Println("Error loading pokedex data:", err)
	} else {
		fmt.Println("Load thành công")
	}

	// Find a Pokemon by its name
	pokemon := findPokemonByName("Pikachu")
	if pokemon != nil {
		fmt.Printf("Found Pokémon: %s %s (HP: %d, Attack: %d)\n", pokemon.Id, pokemon.Name, pokemon.PokeInfo.Hp, pokemon.PokeInfo.Atk)
	} else {
		fmt.Printf("Pokémon with name %s not found.\n", "Pikachu")
	}

	// Find player Pokemons by player's name
	playerName := "thien"
	playerPokemons := findPlayerPokemonByPlayer(playerName)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Pokémons of player %s:\n", playerName)
		for _, pokemon := range playerPokemons {
			fmt.Printf("Pokemon ID: %s, Name: %s, Level: %d, HP: %d\n", pokemon.ID, pokemon.Name, pokemon.Level, pokemon.Hp)
		}
	}

	playerNamePoke := "thien"
	playerPokemonsOne := findPlayerPokemonByPokeID(playerNamePoke, "#001")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Pokémons of player %s:\n", playerNamePoke)

		fmt.Printf("Pokemon ID: %s, Name: %s, Level: %d, HP: %d\n", playerPokemonsOne.ID, playerPokemonsOne.Name, playerPokemonsOne.Level, playerPokemonsOne.Hp)

	}
}

func loadPlayerPokemons(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &playersPokemons)
}

func findPlayerPokemonByPlayer(playerName string) []PlayerPokeInfo {
	var pokemon []PlayerPokeInfo
	for _, p := range playersPokemons {
		if p.Owner == playerName {
			pokemon = p.PlayerPokeInfo
		}
	}
	return pokemon
}

func findPokemonByNameO(name string) *Pokemon {
	for _, p := range pokedex {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

func loadPokedex(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &pokedex) // gán data vào pokedex
}

func findPlayerPokemonByPokeID(playerName string, idPoke string) PlayerPokeInfo {
	var pokemon PlayerPokeInfo
	for _, p := range playersPokemons {
		if p.Owner == playerName {
			for _, po := range p.PlayerPokeInfo {
				if po.ID == idPoke {
					pokemon = po
				}
			}
		}
	}
	return pokemon
}
