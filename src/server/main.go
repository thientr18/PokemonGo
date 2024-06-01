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
	pokedexData = "JSON/pokedex.json"
)

type Pokemon struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
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

var players = make(map[string]*Player)
var battles = make(map[string]*Battle)
var pokedex Pokedex

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
			if !checkExistedClient(parts[1]) {
				sendMessageToClient("duplicated-username", addr, conn)
			} else {
				username := parts[1]
				players[username] = &Player{Name: username, Addr: addr}
				fmt.Printf("User '%s' joined\n", username)
				sendMessageToClient("Welcome to the chat, "+username+"!", addr, conn)
			}
		case "@all":
			if !players[senderName].isInBattle() {
				broadcastMessage(parts[1], senderName, conn) // Pass sender's name
			} else {
				sendMessageToClient("Cannot chat in the battle!\nSend your next action:", addr, conn)
			}
		case "@quit":
			delete(players, senderName)
			fmt.Printf("User '%s' left\n", senderName)
			sendMessageToClient("Goodbye, "+senderName+"!", addr, conn)
		case "@private":
			if !players[senderName].isInBattle() {
				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				recipient := nextPart[0]
				if checkExistedClient(recipient) {
					sendErrorMessageToPlayer("Error: Recipient did not exist in the server!", addr, conn)
					break
				} else {
					privateMessage := senderName + " (private): " + nextPart[1]
					sendPrivateMessage(privateMessage, recipient, conn)
				}
			} else {
				sendMessageToClient("Cannot chat in the battle!\nSend your next action:", addr, conn)
			}
		case "@battle":
			if players[senderName].isInBattle() {
				sendMessageToClient("You are already in a battle!", addr, conn)
				break
			}
			temp := parts[1]
			nextPart := strings.SplitN(temp, " ", 2)
			opponent := nextPart[0]
			if checkExistedClient(opponent) {
				sendErrorMessageToPlayer("Error: Opponent did not exist in the server!", addr, conn)
				break
			}
			if players[opponent].isInBattle() {
				sendErrorMessageToPlayer("Error: Opponent is already in a battle!", addr, conn)
				break
			}
			handleBattle(senderName, opponent, conn)
		case "@pick":
			if !players[senderName].isInBattle() {
				sendMessageToClient("Invalid command", addr, conn)
				break
			}
			temp := parts[1]
			nextPart := strings.SplitN(temp, " ", 2)

			pickPokemon(senderName, nextPart[0], conn)
		default:
			sendMessageToClient("Invalid command", addr, conn)
		}
	} else {
		sendMessageToClient("Invalid command format", addr, conn)
	}
}

func checkExistedClient(username string) bool {
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

func sendMessageToClient(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	_, err := conn.WriteToUDP([]byte(message), addr)
	if err != nil {
		fmt.Println("Error sending message:", err)
	}
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

func sendPrivateMessage(message, username string, conn *net.UDPConn) {
	player := players[username]
	_, err := conn.WriteToUDP([]byte(message), player.Addr)
	if err != nil {
		fmt.Println("Error sending private message:", err)
	}
}

func sendErrorMessageToPlayer(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	_, err := conn.WriteToUDP([]byte(message), addr)
	if err != nil {
		fmt.Println("Error sending error message:", err)
	}
}

func handleBattle(player1, player2 string, conn *net.UDPConn) {
	battleID := player1 + "-" + player2
	battle := &Battle{
		Player1: player1,
		Player2: player2,
		Turn:    player1,
	}
	battles[battleID] = battle
	players[player1].Battle = battle
	players[player2].Battle = battle

	sendMessageToClient("Battle started between "+player1+" and "+player2+"!", players[player1].Addr, conn)
	sendMessageToClient("Battle started between "+player1+" and "+player2+"!", players[player2].Addr, conn)
	sendMessageToClient(player1+" picks first!", players[player1].Addr, conn)
	sendMessageToClient(player1+" picks first!", players[player2].Addr, conn)
}

func (p *Player) isInBattle() bool {
	return p.Battle != nil
}

func pickPokemon(playerName, pokemonID string, conn *net.UDPConn) {
	player := players[playerName]
	if len(player.Pokemons) < 3 {
		for _, p := range pokedex.Pokemons {
			if fmt.Sprintf("%d", p.ID) == pokemonID {
				player.Pokemons = append(player.Pokemons, PlayerPokemon{Name: p.Name, ID: p.ID, Level: 1, Exp: 0})
				sendMessageToClient("You picked "+p.Name, player.Addr, conn)
				break
			}
		}
	} else {
		sendMessageToClient("You have already picked 3 Pokemons!", player.Addr, conn)
	}

	if len(player.Pokemons) == 3 {
		opponent := player.Battle.Player1
		if player.Battle.Player1 == playerName {
			opponent = player.Battle.Player2
		}
		if len(players[opponent].Pokemons) == 3 {
			startBattle(playerName, opponent, conn)
		} else {
			sendMessageToClient("Waiting for opponent to pick Pokemons.", player.Addr, conn)
		}
	}
}

func startBattle(player1, player2 string, conn *net.UDPConn) {
	battleID := player1 + "-" + player2
	battle := battles[battleID]
	sendMessageToClient("Both players have picked 3 Pokemons. Let the battle begin!", players[player1].Addr, conn)
	sendMessageToClient("Both players have picked 3 Pokemons. Let the battle begin!", players[player2].Addr, conn)
	battle.Turn = player1
}

func loadPokedex(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &pokedex)
}

func getPickedPokemons(playerName string) []PlayerPokemon {
	pickedPokemons := make([]PlayerPokemon, 0)
	player, ok := players[playerName]
	if !ok {
		return pickedPokemons
	}
	for _, pokemon := range player.Pokemons {
		pickedPokemons = append(pickedPokemons, pokemon)
	}
	return pickedPokemons
}

/*
the pokemonID in method pickPokemon() is different ti the pokemonID in the pokedex.
pokemonID in pokedex is to let user can check common info of a particular pokemon, you can call it pokedexID.
But the pokemonID in method pickPokemon() is the id of pokemon owned by a player.
Like a player can have many same pokemon so I want each owned pokemon have a unique id to not be duplicated.
*/
