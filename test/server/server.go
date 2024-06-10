package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"sync"
)

type Pokemon struct {
	Name   string `json:"name"`
	HP     int    `json:"hp"`
	Attack int    `json:"atk"`
}

type Player struct {
	ID       string
	Pokemons map[string]Pokemon
	Active   string
}

type Game struct {
	TurnOrder []string
	Current   int
	Players   map[string]*Player
	Status    string // "inviting", "active"
}

type GameState struct {
	mu    sync.Mutex
	Games map[string]*Game
}

var gameState = GameState{
	Games: make(map[string]*Game),
}

func handleConnection(conn *net.UDPConn, addr *net.UDPAddr, msg string) {
	gameState.mu.Lock()
	defer gameState.mu.Unlock()

	parts := strings.Split(msg, "|")
	if len(parts) < 2 {
		return
	}

	cmd, id := parts[0], parts[1]

	switch cmd {
	case "JOIN":
		conn.WriteToUDP([]byte("POKEMON_LIST|"+formatPokemonList()), addr)
	case "CHOOSE":
		if len(parts) < 5 {
			conn.WriteToUDP([]byte("ERROR|Invalid choose command"), addr)
			return
		}
		if _, exists := gameState.Games[id]; !exists {
			pokemons := make(map[string]Pokemon)
			for i := 2; i < 5; i++ {
				for _, p := range availablePokemons {
					if p.Name == parts[i] {
						pokemons[parts[i]] = p
						break
					}
				}
			}
			player := &Player{ID: id, Pokemons: pokemons, Active: parts[2]}
			gameState.Games[id] = &Game{
				TurnOrder: []string{id},
				Current:   0,
				Players:   map[string]*Player{id: player},
				Status:    "waiting",
			}
			conn.WriteToUDP([]byte("CHOSEN|"+id), addr)
		}
	case "INVITE":
		if len(parts) == 3 {
			inviteeID := parts[2]
			if game, exists := gameState.Games[id]; exists && game.Status == "waiting" {
				game.Status = "inviting"
				game.TurnOrder = append(game.TurnOrder, inviteeID)
				conn.WriteToUDP([]byte("INVITED|"+inviteeID), addr)
			} else {
				conn.WriteToUDP([]byte("ERROR|Cannot invite player"), addr)
			}
		}
	case "ACCEPT":
		if len(parts) == 3 {
			inviterID := parts[2]
			if game, ok := gameState.Games[inviterID]; ok && game.Status == "inviting" && len(game.TurnOrder) == 2 {
				player := game.Players[game.TurnOrder[0]]
				game.Players[id] = &Player{ID: id, Pokemons: player.Pokemons, Active: player.Active}
				game.Status = "active"
				conn.WriteToUDP([]byte("START|"+inviterID), addr)
				conn.WriteToUDP([]byte("START|"+id), addr)
			} else {
				conn.WriteToUDP([]byte("ERROR|Invalid invitation"), addr)
			}
		}
	case "ATTACK":
		for _, game := range gameState.Games {
			if player, ok := game.Players[id]; ok {
				if game.TurnOrder[game.Current] != id {
					conn.WriteToUDP([]byte("ERROR|Not your turn"), addr)
					return
				}
				opponentID := game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]
				if opponent, ok := game.Players[opponentID]; ok {
					opponentPokemon := opponent.Pokemons[opponent.Active]
					opponentPokemon.HP -= player.Pokemons[player.Active].Attack
					opponent.Pokemons[opponent.Active] = opponentPokemon // Reassign modified Pokémon back to map
					if opponentPokemon.HP <= 0 {
						conn.WriteToUDP([]byte("WIN|"+id), addr)
					} else {
						conn.WriteToUDP([]byte("ATTACKED|"+id), addr)
						game.Current = (game.Current + 1) % len(game.TurnOrder)
					}
				}
			}
		}
	case "CHANGE":
		for _, game := range gameState.Games {
			if player, ok := game.Players[id]; ok {
				if game.TurnOrder[game.Current] != id {
					conn.WriteToUDP([]byte("ERROR|Not your turn"), addr)
					return
				}
				if len(parts) == 3 {
					newActive := parts[2]
					if _, ok := player.Pokemons[newActive]; ok {
						player.Active = newActive
						conn.WriteToUDP([]byte("CHANGED|"+newActive), addr)
						game.Current = (game.Current + 1) % len(game.TurnOrder)
					} else {
						conn.WriteToUDP([]byte("ERROR|Invalid Pokémon"), addr)
					}
				}
			}
		}
	}
}

var availablePokemons []Pokemon

func loadPokemonData(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var pokemons struct {
		Pokemons []Pokemon `json:"pokemons"`
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
		sb.WriteString(fmt.Sprintf("%s (HP: %d, Attack: %d)\n", p.Name, p.HP, p.Attack))
	}
	return sb.String()
}

func main() {
	if err := loadPokemonData("test\\pokemon_data.json"); err != nil {
		panic(err)
	}

	addr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Println("Server started on port 8080")

	buffer := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		go handleConnection(conn, clientAddr, string(buffer[:n]))
	}
}
