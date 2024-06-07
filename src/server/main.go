package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
)

const (
	HOST        = "localhost"
	PORT        = "8080"
	TYPE        = "udp"
	pokedexData = "src\\JSON\\pokedex.json"
)

type (
	Pokemon struct {
		Name      string `json:"name"`
		PokedexID int    `json:"id"`
	}

	PlayerPokemon []struct {
		Name  string
		ID    string
		Level int
		Exp   int
		Speed int
	}

	Type struct {
		Name   string   `json:"name"`
		Effect []string `json:"effectiveAgainst"`
		Weak   []string `json:"weakAgainst"`
	}

	Pokedex struct {
		Types    []Type    `json:"types"`
		Pokemons []Pokemon `json:"pokemons"`
	}

	Player struct {
		Name                  string
		Addr                  *net.UDPAddr
		Pokemons              []PlayerPokemon
		battleRequestSends    map[string]string // store number of request that a player send: 'map[receivers]sender'
		battleRequestReceives map[string]string // store number of request that a player get: 'map[senders]receiver'
	}

	Battle struct {
		Player1    string
		Player2    string
		Turn       string
		P1Pokemons []PlayerPokemon
		P2Pokemons []PlayerPokemon
	}
)

var pokedex Pokedex

var players = make(map[string]*Player)

var battles = make(map[string]*Battle)

var p1pokemons = make(map[string]*PlayerPokemon)
var p2pokemons = make(map[string]*PlayerPokemon)

func main() {
	// Load the pokedex data from the JSON file
	err := loadPokedex(pokedexData)
	if err != nil {
		fmt.Println("Error loading pokedex data:", err)
		return
	}

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

	fmt.Println("Pokemon game has been running on", udpAddr)

	buffer := make([]byte, 1024)

	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading:", err)
			continue
		}

		message := string(buffer[:n])
		handleMessage(message, addr, conn)
	}
}

func handleMessage(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	if strings.HasPrefix(message, "@") {
		parts := strings.SplitN(message, " ", 2)
		command := parts[0]
		senderName := getPlayernameByAddr(addr) // Get sender's name

		switch command {
		case "@join":
			if !checkExistedPlayer(parts[1]) {
				sendMessage("duplicated_username", addr, conn)
			} else {
				username := parts[1]
				players[username] = &Player{Name: username,
					Addr:                  addr,
					battleRequestSends:    make(map[string]string),
					battleRequestReceives: make(map[string]string),
				}
				fmt.Printf("User '%s' joined\n", username)
				sendMessage("Welcome to the chat, "+username+"!", addr, conn)
			}
		case "@all":
			if !players[senderName].isInBattle() {
				broadcastMessage(parts[1], senderName, conn) // Pass sender's name
			} else {
				sendMessage("Cannot chat in the battle!\nSend your next action:", addr, conn)
			}
		case "@quit":
			delete(players, senderName)
			fmt.Printf("User '%s' left\n", senderName)
			sendMessage("Goodbye, "+senderName+"!", addr, conn)
			// surrentder()
		case "@private":
			if !players[senderName].isInBattle() {
				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				receiver := nextPart[0]
				if checkExistedPlayer(receiver) {
					sendMessage("Error: Receiver did not exist in the server!", addr, conn)
					break
				} else {
					privateMessage := senderName + " (private): " + nextPart[1]
					sendMessage(privateMessage, players[receiver].Addr, conn)
				}
			} else {
				sendMessage("Cannot chat in the battle!\nSend your next action:", addr, conn)
			}
		case "@battle":
			if isInBattle(senderName) {
				sendMessage("You are already in a battle!", addr, conn)
				break
			}
			temp := parts[1]
			nextPart := strings.SplitN(temp, " ", 2)
			opponent := nextPart[0]
			if checkExistedPlayer(opponent) {
				sendMessage("Error: Opponent did not exist in the server!", addr, conn)
				break
			}
			if isInBattle(opponent) {
				sendMessage("Error: Opponent is already in a battle!", addr, conn)
				break
			}

			players[senderName].battleRequestSends[opponent] = senderName
			players[opponent].battleRequestReceives[senderName] = opponent

			battleRequestMessage := "Player '" + senderName + "' requests you a pokemon battle!"
			sendMessage(battleRequestMessage, players[opponent].Addr, conn)
		case "@accept":
			if isInBattle(senderName) {
				sendMessage("You are already in a battle!", addr, conn)
				break
			}
			temp := parts[1]
			nextPart := strings.SplitN(temp, " ", 2)
			opponent := nextPart[0]
			if players[senderName].battleRequestReceives[opponent] == senderName &&
				players[opponent].battleRequestSends[senderName] == opponent {
				sendMessage("You accepted a battle with player '"+opponent+"'", addr, conn)
				sendMessage("Battle Started!", addr, conn)
				sendMessage("Your battle request with player '"+senderName+"' is accepted!", players[opponent].Addr, conn)
				sendMessage("Battle Started!", players[opponent].Addr, conn)

				battleHandler(senderName, opponent, addr, players[opponent].Addr)
			} else {
				sendMessage("Invalid acception! (WRONG opppent name or NOT RECEIVES battle request from this opponent)", addr, conn)
			}

		case "@deny":
			temp := parts[1]
			nextPart := strings.SplitN(temp, " ", 2)
			opponent := nextPart[0]

			if players[senderName].battleRequestReceives[opponent] == senderName &&
				players[opponent].battleRequestSends[senderName] == opponent {
				delete(players[opponent].battleRequestSends, senderName)
				delete(players[senderName].battleRequestReceives, opponent)

				sendMessage("You denied a battle with player '"+opponent+"'", addr, conn)
				sendMessage("Your battle request to player '"+senderName+"' was dinied!", players[opponent].Addr, conn)
			} else {
				sendMessage("Invalid acception! (WRONG opppent name or NOT RECEIVES battle request from this opponent)", addr, conn)
			}
		// case "@pick":
		// 	if !players[senderName].isInBattle() {
		// 		sendMessage("Invalid command", senderName, conn)
		// 		break
		// 	}
		// 	temp := parts[1]
		// 	nextPart := strings.SplitN(temp, " ", 2)
		// 	pickPokemon(senderName, nextPart[0], conn)
		// case "@attack":
		// 	if players[senderName].isInBattle() {
		// 		attack()
		// 	}
		// case "@change":
		// 	if players[senderName].isInBattle() {
		// 		temp := parts[1]
		// 		nextPart := strings.SplitN(temp, " ", 2)
		// 		pokemonID := nextPart[0]
		// 		changePokemon(pokemonID)
		// 	}
		// case "@surrender":
		// 	handleSurrender(senderName, conn)

		default:
			sendMessage("Invalid command", addr, conn)
		}
	} else {
		sendMessage("Invalid command format", addr, conn)
	}
}

func isInBattle(p string) bool {
	_, exists := battles[p]
	if !exists {
		return true
	} else {
		return false
	}
}

func battleHandler(player1 string, player2 string, addr1 *net.UDPAddr, addr2 *net.UDPAddr) {
	var battleTemp = "ddddd"

	battles[battleTemp] = &Battle{
		Player1:    string,
		Player2:    string,
		Turn:       string,
		P1Pokemons: []PlayerPokemon,
		P2Pokemons: []PlayerPokemon}

	// chooseThreePokemons()
	checkFirstTurn(addr1, addr2)
}

func chooseThreePokemons(player1 *net.UDPAddr) {

}

func checkFirstTurn(player1 *net.UDPAddr, player2 *net.UDPAddr) {

}

func broadcastMessage(message string, senderName string, conn *net.UDPConn) {
	for username, player := range players {
		if username != senderName {
			fullMessage := senderName + " (public): " + message // Include sender's name
			_, err := conn.WriteToUDP([]byte(fullMessage), player.Addr)
			if err != nil {
				fmt.Println("Error broadcasting message:", err)
			}
		}
	}
}

func sendMessage(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	_, err := conn.WriteToUDP([]byte(message), addr)
	if err != nil {
		fmt.Println("Error sending error message:", err)
	}
}

func checkExistedPlayer(username string) bool {
	_, exists := players[username]
	if !exists {
		return true
	} else {
		return false
	}
}

func getPlayernameByAddr(addr *net.UDPAddr) string {
	for _, player := range players {
		if player.Addr.IP.Equal(addr.IP) && player.Addr.Port == addr.Port {
			return player.Name
		}
	}
	return ""
}

func loadPokedex(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &pokedex)
}
