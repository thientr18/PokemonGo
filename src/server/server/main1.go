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
		return
	}

	// Load the pokedex data from the JSON file
	// if err := loadPokemonData("test\\pokemon_data.json"); err != nil {
	// 	panic(err)
	// }

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

	// Initialize player1 and player2
	initializePlayers(conn)

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

func initializePlayers(conn *net.UDPConn) {
	addr1, err := net.ResolveUDPAddr(TYPE, HOST+":8081")
	if err != nil {
		fmt.Println("Error resolving UDP address for player1:", err)
		return
	}
	addr2, err := net.ResolveUDPAddr(TYPE, HOST+":8082")
	if err != nil {
		fmt.Println("Error resolving UDP address for player2:", err)
		return
	}

	player1 := &Player{
		Name:                  "player1",
		Addr:                  addr1,
		battleRequestSends:    make(map[string]string),
		battleRequestReceives: make(map[string]string),
	}
	player2 := &Player{
		Name:                  "player2",
		Addr:                  addr2,
		battleRequestSends:    make(map[string]string),
		battleRequestReceives: make(map[string]string),
	}

	players[player1.Name] = player1
	players[player2.Name] = player2

	// Automatically start a battle between player1 and player2
	startBattle(player1, player2, conn)
}

func startBattle(player1, player2 *Player, conn *net.UDPConn) {
	gameState.mu.Lock()
	defer gameState.mu.Unlock()

	gameState.Battles[player1.Name] = &Battle{
		Players:        map[string]*Player{player1.Name: player1, player2.Name: player2},
		ActivePokemons: make(map[string]BattlePokemon),
		TurnOrder:      []string{player1.Name, player2.Name},
		Current:        0,
		Status:         "waiting",
	}

	inBattleWith[player1.Name] = player2.Name
	inBattleWith[player2.Name] = player1.Name

	sendMessage("Battle started between player1 and player2!", player1.Addr, conn)
	sendMessage("Battle started between player1 and player2!", player2.Addr, conn)
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
				sendMessage("Automatic join is disabled in this mode", addr, conn)
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
					sendMessage(privateMessage, players[receiver].Addr, conn)
				}
			case "@battle":
				sendMessage("Automatic battle initiation is enabled in this mode", addr, conn)
			case "@accept":
				sendMessage("Automatic battle acceptance is enabled in this mode", addr, conn)
			case "@deny":
				sendMessage("Automatic battle denial is enabled in this mode", addr, conn)
			case "@list":
				sendMessage("@pokemon_list"+formatPokemonList(), addr, conn)
			default:
				sendMessage("Invalid command 1", addr, conn)
			}
		} else {
			switch command {
			case "@all":
				sendMessage("Cannot chat all in the battle!\nSend your next action:", addr, conn)
			case "@quit":
				delete(players, senderName)
				fmt.Printf("User '%s' left\n", senderName)
				sendMessage("Goodbye '"+senderName+"'!", addr, conn)
				// surrender()
			case "@private":
				if len(parts) < 2 {
					sendMessage("Invalid command", addr, conn)
					break
				}

				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				receiver := nextPart[0]
				if receiver != inBattleWith[senderName] {
					sendMessage("You cannot send private message to other player except your opponent in battle!\nSend your next action:", addr, conn)
					break
				}
				if !checkExistedPlayer(receiver) {
					sendMessage("Error: Receiver did not exist in the server!", addr, conn)
					break
				} else {
					privateMessage := senderName + " (private): " + nextPart[1]
					sendMessage(privateMessage, players[receiver].Addr, conn)
				}
			case "@accept":
				sendMessage("You are already in battle!\nSend your next action:", addr, conn)
			case "@deny":
				sendMessage("You are already in battle!\nSend your next action:", addr, conn)
			case "@battle":
				sendMessage("You are already in battle!\nSend your next action:", addr, conn)
			case "@list":
				sendMessage("@pokemon_list"+formatPokemonList(), addr, conn)
			default:
				sendMessage("Invalid command 2", addr, conn)
			}
		}
	}
}

func getPlayernameByAddr(addr *net.UDPAddr) string {
	for name, player := range players {
		if player.Addr.String() == addr.String() {
			return name
		}
	}
	return ""
}

func isInBattle(name string) bool {
	_, exists := inBattleWith[name]
	return exists
}

func broadcastMessage(message, senderName string, conn *net.UDPConn) {
	for name, player := range players {
		if name != senderName {
			sendMessage(senderName+": "+message, player.Addr, conn)
		}
	}
}

func sendMessage(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	_, err := conn.WriteToUDP([]byte(message), addr)
	if err != nil {
		fmt.Println("Error sending message:", err)
	}
}

func loadPokedex(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var pokemons []Pokemon
	if err := json.Unmarshal(data, &pokemons); err != nil {
		return err
	}

	return nil
}

func loadPokemonData(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &availablePokemons); err != nil {
		return err
	}

	return nil
}

func checkExistedPlayer(name string) bool {
	_, exists := players[name]
	return exists
}

func formatPokemonList() string {
	var sb strings.Builder
	sb.WriteString("\nAvailable Pokemons:\n")
	for _, p := range availablePokemons {
		sb.WriteString(fmt.Sprintf("- %s (Level %d)\n", p.Name, p.Level))
	}
	return sb.String()
}
