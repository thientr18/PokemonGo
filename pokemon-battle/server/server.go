package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
)

// Define Pokémon struct
type Pokemon struct {
	Name    string
	HP      int
	Attack  int
	Defense int
	Speed   int
}

// Define Player struct
type Player struct {
	ID      int
	Pokemon Pokemon
	Addr    *net.UDPAddr
}

// Define Game State struct
type GameState struct {
	Players []Player
	Turn    int
}

// Define a struct for messages
type Message struct {
	PlayerID int
	Action   string
}

// Global game state
var gameState GameState

func main() {
	addr := net.UDPAddr{
		Port: 8080,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Server started on port 8080")

	gameState = GameState{
		Players: []Player{},
		Turn:    0,
	}

	buf := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		var msg Message
		err = json.Unmarshal(buf[:n], &msg)
		if err != nil {
			log.Println(err)
			continue
		}

		handleClientMessage(conn, clientAddr, msg)
	}
}

func handleClientMessage(conn *net.UDPConn, addr *net.UDPAddr, msg Message) {
	switch msg.Action {
	case "join":
		addPlayer(addr)
	case "attack":
		processAttack(msg.PlayerID)
	}
	sendGameState(conn)
}

func addPlayer(addr *net.UDPAddr) {
	if len(gameState.Players) >= 2 {
		return
	}
	newPlayer := Player{
		ID: len(gameState.Players) + 1,
		Pokemon: Pokemon{
			Name:    "Pikachu",
			HP:      100,
			Attack:  50,
			Defense: 40,
			Speed:   90,
		},
		Addr: addr,
	}
	gameState.Players = append(gameState.Players, newPlayer)
}

func processAttack(playerID int) {
	if gameState.Turn != playerID {
		return
	}
	opponentID := (playerID % 2) + 1
	for i := range gameState.Players {
		if gameState.Players[i].ID == opponentID {
			damage := gameState.Players[playerID-1].Pokemon.Attack - gameState.Players[i].Pokemon.Defense
			if damage > 0 {
				gameState.Players[i].Pokemon.HP -= damage
			}
		}
	}
	gameState.Turn = opponentID
}

func sendGameState(conn *net.UDPConn) {
	for _, player := range gameState.Players {
		stateJSON, _ := json.Marshal(gameState)
		conn.WriteToUDP(stateJSON, player.Addr)
	}
}
