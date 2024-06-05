package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

type Pokemon struct {
	Name    string
	HP      int
	Attack  int
	Defense int
	Speed   int
}

type Player struct {
	ID      int
	Pokemon Pokemon
}

type GameState struct {
	Players []Player
	Turn    int
}

type Message struct {
	PlayerID int
	Action   string
}

func main() {
	serverAddr := net.UDPAddr{
		Port: 8080,
		IP:   net.ParseIP("127.0.0.1"),
	}

	conn, err := net.DialUDP("udp", nil, &serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Connected to the server")

	// Join the game
	joinMsg := Message{Action: "join"}
	sendMessage(conn, joinMsg)

	go listenForUpdates(conn)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter command: ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		if text == "attack" {
			msg := Message{Action: "attack"}
			sendMessage(conn, msg)
		}
	}
}

func sendMessage(conn *net.UDPConn, msg Message) {
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
		return
	}
	conn.Write(msgJSON)
}

func listenForUpdates(conn *net.UDPConn) {
	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		var gameState GameState
		err = json.Unmarshal(buf[:n], &gameState)
		if err != nil {
			log.Println(err)
			continue
		}

		displayGameState(gameState)
	}
}

func displayGameState(gameState GameState) {
	fmt.Println("Game State:")
	for _, player := range gameState.Players {
		fmt.Printf("Player %d: %s (HP: %d)\n", player.ID, player.Pokemon.Name, player.Pokemon.HP)
	}
	fmt.Printf("Current Turn: Player %d\n", gameState.Turn)
}
