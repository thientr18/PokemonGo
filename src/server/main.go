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
			if !checkExistedPlayer(parts[1]) {
				sendMessage("duplicated-username", senderName, conn)
			} else {
				username := parts[1]
				players[username] = &Player{Name: username, Addr: addr}
				fmt.Printf("User '%s' joined\n", username)
				sendMessage("Welcome to the chat, "+username+"!", username, conn)
			}
		case "@all":
			if !players[senderName].isInBattle() {
				broadcastMessage(parts[1], senderName, conn) // Pass sender's name
			} else {
				sendMessage("Cannot chat in the battle!\nSend your next action:", senderName, conn)
			}
		case "@quit":
			delete(players, senderName)
			fmt.Printf("User '%s' left\n", senderName)
			sendMessage("Goodbye, "+senderName+"!", senderName, conn)
			// surrentder()
		case "@private":
			if !players[senderName].isInBattle() {
				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				recipient := nextPart[0]
				if checkExistedPlayer(recipient) {
					sendMessage("Error: Recipient did not exist in the server!", senderName, conn)
					break
				} else {
					privateMessage := senderName + " (private): " + nextPart[1]
					sendMessage(privateMessage, recipient, conn)
				}
			} else {
				sendMessage("Cannot chat in the battle!\nSend your next action:", senderName, conn)
			}
		case "@battle":
			if players[senderName].isInBattle() {
				sendMessage("You are already in a battle!", senderName, conn)
				break
			}
			temp := parts[1]
			nextPart := strings.SplitN(temp, " ", 2)
			opponent := nextPart[0]
			if checkExistedPlayer(opponent) {
				sendMessage("Error: Opponent did not exist in the server!", senderName, conn)
				break
			}
			if players[opponent].isInBattle() {
				sendMessage("Error: Opponent is already in a battle!", senderName, conn)
				break
			}
			battleRequest := "Player '" + senderName + "' requests you a pokemon battle!"
			sendMessage(battleRequest, opponent, conn)
		case "@accept":
			temp := parts[1]
			nextPart := strings.SplitN(temp, " ", 2)
			opponent := nextPart[0]
			startBattle(senderName, opponent, conn)
		case "@deny":
			temp := parts[1]
			nextPart := strings.SplitN(temp, " ", 2)
			opponent := nextPart[0]
			deniedMessage := "Your battle request to  '" + opponent + "' was denied!"
			sendMessage(deniedMessage, opponent, conn)
		case "@pick":
			if !players[senderName].isInBattle() {
				sendMessage("Invalid command", senderName, conn)
				break
			}
			temp := parts[1]
			nextPart := strings.SplitN(temp, " ", 2)
			pickPokemon(senderName, nextPart[0], conn)
		case "@surrender":
			handleSurrender(senderName, conn)
		default:
			sendMessage("Invalid command", senderName, conn)
		}
	} else {
		sendMessage("Invalid command format", getPlayernameByAddr(addr), conn)
	}
}

func (p *Player) isInBattle() bool {
	return p.Battle != nil
}

func startBattle(player1, player2 string, conn *net.UDPConn) {
	battleID := player1 + "-" + player2
	battle := battles[battleID]
	sendMessage("Both players have picked 3 Pokemons. Let the battle begin!", player1, conn)
	sendMessage("Both players have picked 3 Pokemons. Let the battle begin!", player2, conn)
	battle.Turn = player1
}

func pickPokemon(playerName, pokemonID string, conn *net.UDPConn) {
	player := players[playerName]
	if len(player.Pokemons) < 3 {
		for _, p := range player.Pokemons {
			if fmt.Sprintf("%s", p.ID) == pokemonID {
				player.Pokemons = append(player.Pokemons, PlayerPokemon{Name: p.Name, ID: p.ID, Level: 1, Exp: 0}) // batle.pokemon = append
				sendMessage("You picked "+p.Name, playerName, conn)
				break
			}
		}
	} else {
		sendMessage("You have already picked 3 Pokemons!", playerName, conn)
	}

	if len(player.Pokemons) == 3 {
		opponent := player.Battle.Player1
		if player.Battle.Player1 == playerName {
			opponent = player.Battle.Player2
		}
		if len(players[opponent].Pokemons) == 3 {
			startBattle(playerName, opponent, conn)
		} else {
			sendMessage("Waiting for opponent to pick Pokemons.", playerName, conn)
		}
	}
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

func checkSpeed(pokemon1, pokemon2 string, conn *net.UDPConn) string {

	return ""
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

	sendMessage("Battle started between "+player1+" and "+player2+"!", player1, conn)
	sendMessage("Battle started between "+player1+" and "+player2+"!", player2, conn)
	sendMessage(player1+" picks first pokemon!", player1, conn)
	sendMessage(player1+" picks first pokemon!", player2, conn)
}

func handleSurrender(playerName string, conn *net.UDPConn) {
	//retrieve player surrender
	player := players[playerName]

	//if the player is in the battle
	if player.Battle == nil {
		sendMessage("You are not in a battle!", playerName, conn)
		return
	}

	//the opposite's name, but if the surrender player is player 1, the opposite is player 2
	opponentName := player.Battle.Player1
	if player.Battle.Player1 == playerName {
		opponentName = player.Battle.Player2
	}

	//Calculate the total accumulated exp on losing team
	totalExp := 0
	for _, pokemon := range player.Pokemons {
		totalExp += pokemon.Exp
	}

	expShare := totalExp / 3

	//add the expShare to win team
	for i := range players[opponentName].Pokemons {
		players[opponentName].Pokemons[i].Exp += expShare
	}

	//Notify the surrender player and opponent
	sendMessage("You surrendered! "+opponentName+" win the battle", playerName, conn)
	sendMessage(playerName+"surrendered ! You win the battle! ", opponentName, conn)

	//Clear the battle state
	player.Battle = nil
	players[opponentName].Battle = nil

	delete(battles, player.Battle.Player1+" - "+player.Battle.Player2)
}

func loadPokedex(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &pokedex)
}

/*------------------------------------------------------------------------------------------------------------------------------------------------------------------*/
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

func sendMessage(message, username string, conn *net.UDPConn) {
	player := players[username]
	_, err := conn.WriteToUDP([]byte(message), player.Addr)
	if err != nil {
		fmt.Println("Error sending private message:", err)
	}
}
