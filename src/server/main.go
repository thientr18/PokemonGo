package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"sync"
)

const (
	HOST        = "localhost"
	PORT        = "8080"
	TYPE        = "udp"
	pokedexData = "src\\database\\pokedex.json"
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
		Addr                  *net.UDPAddr
		Pokemons              map[string]PlayerPokemon
		BattlePokemon         map[string]BattlePokemon
		battleRequestSends    map[string]string // store number of request that a player send: 'map[receivers]sender'
		battleRequestReceives map[string]string // store number of request that a player get: 'map[senders]receiver'
		Active                string
	}

	BattlePokemon struct {
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

	Battle struct {
		Players        map[string]*Player
		ActivePokemons map[string]BattlePokemon // Store active Pokemons in the battle
		TurnOrder      []string
		Current        int
		Status         string // "waiting", "inviting", "active"
	}

	GameState struct {
		mu      sync.Mutex
		Battles map[string]*Battle
		Players map[string]*Player
	}
)

var gameState = GameState{
	Battles: make(map[string]*Battle),
	Players: make(map[string]*Player),
}

var pokedex PokeInfo // pokedex

var players = make(map[string]*Player) // list of player

var inBattleWith = make(map[string]string) // check player is in battle or not

var availablePokemons []PlayerPokemon // store pokemons of player | load data failed

var BattlePokemons []BattlePokemon // chưa load data

func main() {
	// Load the pokedex data from the JSON file
	err := loadPokedex(pokedexData)
	if err != nil {
		fmt.Println("Error loading pokedex data:", err)

	}

	// Load the pokedex data from the JSON file
	if err := loadPokemonData("test\\pokemon_data.json"); err != nil {
		panic(err)
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
		go handleMessage(message, addr, conn)
	}
}

func handleMessage(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	gameState.mu.Lock()
	defer gameState.mu.Unlock()

	fmt.Println(message)

	if strings.HasPrefix(message, "@") {
		parts := strings.SplitN(message, " ", 2)
		command := parts[0]
		senderName := getPlayernameByAddr(addr) // Get sender's name

		if !isInBattle(senderName) {
			switch command {
			case "@join":
				if len(parts) < 2 {
					sendMessage("Invalid command", addr, conn)
					break
				}
				if checkExistedPlayer(parts[1]) {
					sendMessage("duplicated_username", addr, conn)
				} else if checkExistedPlayerByAddr(addr) {
					sendMessage("Your address are exsisting in the server", addr, conn)
				} else {
					username := parts[1]
					players[username] = &Player{
						Name:                  username,
						Addr:                  addr,
						battleRequestSends:    make(map[string]string),
						battleRequestReceives: make(map[string]string),
					}
					fmt.Printf("User '%s' joined\n", username)
					sendMessage("Welcome to the chat '"+username+"'!", addr, conn)
				}
			case "@all":
				broadcastMessage(parts[1], senderName, conn) // Pass sender's name
			case "@quit":
				delete(players, senderName)
				fmt.Printf("User '%s' left\n", senderName)
				sendMessage("Goodbye '"+senderName+"'!", addr, conn)
			case "@private":
				if len(parts) < 2 {
					sendMessage("Invalid command", addr, conn)
					break
				}

				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				receiver := nextPart[0]
				if !checkExistedPlayer(receiver) {
					sendMessage("Error: Receiver did not exist in the server!", addr, conn)
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
	player := players[playerName]
	if player.Battle == nil {
		sendMessage("You are not in a battle!", playerName, conn)
		return
	}

	opponentName := player.Battle.Player1
	if player.Battle.Player1 == playerName {
		opponentName = player.Battle.Player2
	}

	totalExp := 0
	for _, p := range player.Pokemons {
		totalExp += p.Exp
	}
	expShare := totalExp / 3

	for i := range players[opponentName].Pokemons {
		players[opponentName].Pokemons[i].Exp += expShare
	}

	sendMessage("You surrendered! "+opponentName+" wins the battle!", playerName, conn)
	sendMessage(playerName+" surrendered! You win the battle!", opponentName, conn)

	player.Battle = nil
	players[opponentName].Battle = nil
	delete(battles, player.Battle.Player1+"-"+player.Battle.Player2)
}

func loadPokemonData(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var pokemons struct {
		Pokemons []PlayerPokemon `json:"playerpokemons"`
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

func isInBattle(p string) bool {
	_, exists := inBattleWith[p]
	if !exists {
		return false
	} else {
		return true
	}
}

func (p *Player) chooseThreePokemons(pokemon string) {

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
		return false
	} else {
		return true
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

func checkExistedPlayerByAddr(addr *net.UDPAddr) bool {
	for _, player := range players {
		if player.Addr.IP.Equal(addr.IP) && player.Addr.Port == addr.Port {
			return true
		}
	}
	return false
}

func loadPokedex(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &pokedex)
}
