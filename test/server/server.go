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
	Attack int    `json:"attack"`
}

type BattlePokemon struct {
	Name   string
	HP     int
	Attack int
}

type Player struct {
	ID       string
	Pokemons map[string]Pokemon // Store all Pokemons of a player
}

type Battle struct {
	Players        map[string]*Player
	ActivePokemons map[string]BattlePokemon // Store active Pokemons in the battle
	TurnOrder      []string
	Current        int
	Status         string // "waiting", "inviting", "active"
}

type GameState struct {
	mu      sync.Mutex
	Battles map[string]*Battle
	Players map[string]*Player
}

var gameState = GameState{
	Battles: make(map[string]*Battle),
	Players: make(map[string]*Player),
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
		if _, exists := gameState.Players[id]; !exists {
			gameState.Players[id] = &Player{ID: id, Pokemons: make(map[string]Pokemon)}
			for _, p := range availablePokemons {
				gameState.Players[id].Pokemons[p.Name] = p
			}
		}
		conn.WriteToUDP([]byte("JOINED|"+id), addr)
	case "INVITE":
		if len(parts) == 3 {
			inviteeID := parts[2]
			if game, exists := gameState.Battles[id]; exists && game.Status == "waiting" {
				if len(game.Players) >= 2 {
					conn.WriteToUDP([]byte("ERROR|Battle already has 2 players"), addr)
					return
				}
				game.Status = "inviting"
				game.TurnOrder = append(game.TurnOrder, inviteeID)
				conn.WriteToUDP([]byte("INVITED|"+inviteeID), addr)
			} else {
				gameState.Battles[id] = &Battle{
					TurnOrder:      []string{id, inviteeID},
					Current:        0,
					Players:        map[string]*Player{},
					ActivePokemons: map[string]BattlePokemon{},
					Status:         "inviting",
				}
				conn.WriteToUDP([]byte("INVITED|"+inviteeID), addr)
			}
		}
	case "ACCEPT":
		if len(parts) == 3 {
			inviterID := parts[2]
			if game, ok := gameState.Battles[inviterID]; ok && game.Status == "inviting" {
				if len(game.Players) >= 2 {
					conn.WriteToUDP([]byte("ERROR|Battle already has 2 players"), addr)
					return
				}
				game.Players[id] = gameState.Players[id]
				game.Status = "waiting"
				conn.WriteToUDP([]byte("ACCEPTED|"+inviterID), addr)
			} else {
				conn.WriteToUDP([]byte("ERROR|Invalid invitation"), addr)
			}
		}
	case "DENY":
		if len(parts) == 3 {
			inviterID := parts[2]
			if game, ok := gameState.Battles[inviterID]; ok && game.Status == "inviting" {
				delete(gameState.Battles, inviterID)
				conn.WriteToUDP([]byte("DENIED|"+id), addr)
			} else {
				conn.WriteToUDP([]byte("ERROR|Invalid invitation"), addr)
			}
		}
	case "CHOOSE":
		if len(parts) < 5 {
			conn.WriteToUDP([]byte("ERROR|Invalid choose command"), addr)
			return
		}
		if game, exists := gameState.Battles[id]; exists && game.Status == "waiting" {
			for i := 2; i < 5; i++ {
				chosen := parts[i]
				if p, ok := gameState.Players[id].Pokemons[chosen]; ok {
					game.ActivePokemons[id+"_"+chosen] = BattlePokemon{Name: p.Name, HP: p.HP, Attack: p.Attack}
				} else {
					conn.WriteToUDP([]byte("ERROR|Invalid Pokémon selection"), addr)
					return
				}
			}
			if len(game.ActivePokemons) == 6 { // Both players have chosen their Pokémon
				game.Status = "active"
				conn.WriteToUDP([]byte("START|"+id), addr)
			} else {
				conn.WriteToUDP([]byte("CHOSEN|"+id), addr)
			}
		} else {
			conn.WriteToUDP([]byte("ERROR|No active game found"), addr)
		}
	case "ATTACK":
		for _, game := range gameState.Battles {
			if _, ok := game.Players[id]; ok {
				if game.TurnOrder[game.Current] != id {
					conn.WriteToUDP([]byte("ERROR|Not your turn"), addr)
					return
				}
				opponentID := game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]
				if _, ok := game.Players[opponentID]; ok {
					activePlayer := game.ActivePokemons[id+"_"+game.TurnOrder[game.Current]]
					activeOpponent := game.ActivePokemons[opponentID+"_"+game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]]
					activeOpponent.HP -= activePlayer.Attack
					game.ActivePokemons[opponentID+"_"+game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]] = activeOpponent
					if activeOpponent.HP <= 0 {
						conn.WriteToUDP([]byte("WIN|"+id), addr)
					} else {
						conn.WriteToUDP([]byte("ATTACKED|"+id), addr)
						game.Current = (game.Current + 1) % len(game.TurnOrder) // Change turn after attack
					}
				}
			}
		}
	case "CHANGE":
		for _, game := range gameState.Battles {
			if player, ok := game.Players[id]; ok {
				if game.TurnOrder[game.Current] != id {
					conn.WriteToUDP([]byte("ERROR|Not your turn"), addr)
					return
				}
				if len(parts) == 3 {
					newActive := parts[2]
					if _, ok := player.Pokemons[newActive]; ok {
						game.TurnOrder[game.Current] = newActive
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

func formatPokemonList() string {
	var sb strings.Builder
	for _, p := range availablePokemons {
		sb.WriteString(fmt.Sprintf("%s (HP: %d, Attack: %d),", p.Name, p.HP, p.Attack))
	}
	return sb.String()
}

func main() {
	if err := loadPokemonData("pokemon_data.json"); err != nil {
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
