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

	Player struct {
		Name                  string
		Addr                  *net.UDPAddr
		Pokemons              map[string]PlayerPokemon
		battleRequestSends    map[string]string // store number of request that a player send: 'map[receivers]sender'
		battleRequestReceives map[string]string // store number of request that a player get: 'map[senders]receiver'
		inBattleWith          string
		Active                string
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

	BattlePokemon struct { // 3 pokemons to choose in a battle
		Name  string `json:"Name"`
		Hp    int    `json:"HP"`
		Atk   int    `json:"ATK"`
		Def   int    `json:"DEF"`
		SpAtk int    `json:"Sp.Atk"`
		SpDef int    `json:"Sp.Def"`
		Speed int    `json:"Speed"`
	}

	Game struct {
		TurnOrder []string
		Current   int
		Players   map[string]*Player
		Status    string // "inviting", "active"
	}

	GameState struct {
		mu    sync.Mutex
		Games map[string]*Game
	}
)

var gameState = GameState{
	Games: make(map[string]*Game),
}

var pokedex PokeInfo

var players = make(map[string]*Player)

var inBattlePlayers = make(map[string]string)

var availablePokemons []PlayerPokemon

func main() {
	// Load the pokedex data from the JSON file
	err := loadPokedex(pokedexData)
	if err != nil {
		fmt.Println("Error loading pokedex data:", err)
		return
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
	if strings.HasPrefix(message, "@") {
		parts := strings.SplitN(message, " ", 2)
		command := parts[0]
		senderName := getPlayernameByAddr(addr) // Get sender's name

		if !isInBattle(senderName) {
			switch command {
			case "@join":
				if checkExistedPlayer(parts[1]) {
					sendMessage("duplicated_username", addr, conn)
				} else if checkExistedPlayerByAddr(addr) {
					break
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
				broadcastMessage(parts[1], senderName, conn) // Pass sender's name
			case "@quit":
				delete(players, senderName)
				fmt.Printf("User '%s' left\n", senderName)
				sendMessage("Goodbye, "+senderName+"!", addr, conn)
			case "@private":
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
				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				opponent := nextPart[0]
				if !checkExistedPlayer(opponent) {
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
				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				opponent := nextPart[0]
				if players[senderName].battleRequestReceives[opponent] == senderName &&
					players[opponent].battleRequestSends[senderName] == opponent {
					sendMessage("You accepted a battle with player '"+opponent+"'", addr, conn)
					sendMessage("Battle Started!", addr, conn)
					sendMessage("Choose your first pokemon:", addr, conn)
					sendMessage("Your battle request with player '"+senderName+"' is accepted!", players[opponent].Addr, conn)
					sendMessage("Battle Started!", players[opponent].Addr, conn)
					sendMessage("Choose your first pokemon:", players[opponent].Addr, conn)

					inBattlePlayers[senderName] = opponent
					inBattlePlayers[opponent] = senderName
					players[senderName] = &Player{inBattleWith: opponent}
					players[opponent] = &Player{inBattleWith: senderName}

					conn.WriteToUDP([]byte("POKEMON_LIST|"+formatPokemonList()), addr)
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
			default:
				sendMessage("Invalid command", addr, conn)
			}
		} else {
			switch command {
			case "@all":
				sendMessage("Cannot chat all in the battle!\nSend your next action:", addr, conn)
			case "@quit":
				delete(players, senderName)
				fmt.Printf("User '%s' left\n", senderName)
				sendMessage("Goodbye, "+senderName+"!", addr, conn)
				// surrentder()
			case "@private":
				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				receiver := nextPart[0]
				if receiver != players[senderName].inBattleWith {
					sendMessage("Cannot chat with other players!", addr, conn)
					break
				} else {
					privateMessage := senderName + " (private): " + nextPart[1]
					sendMessage(privateMessage, players[receiver].Addr, conn)
				}
			case "@battle":
				sendMessage("You are already in a battle!", addr, conn)
				break
			case "@accept":
				sendMessage("You are already in a battle!", addr, conn)
				break
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
			default:
				sendMessage("Invalid command", addr, conn)

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
			}
		}
	}
	if strings.HasPrefix(message, "|") {
		gameState.mu.Lock()
		defer gameState.mu.Unlock()
		parts := strings.Split(message, "|")

		senderName := getPlayernameByAddr(addr) // Get sender's name

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
				pokemons := make(map[string]PlayerPokemon)
				for i := 2; i < 5; i++ {
					for _, p := range availablePokemons {
						if p.Name == parts[i] {
							pokemons[parts[i]] = p
							break
						}
					}
				}
				battlePlayer := &Player{Name: senderName, Pokemons: pokemons, Active: parts[2]}
				gameState.Games[id] = &Game{
					TurnOrder: []string{id},
					Current:   0,
					Players:   map[string]*Player{id: battlePlayer},
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
					game.Players[id] = &Player{Name: senderName, Pokemons: player.Pokemons, Active: player.Active}
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
						opponentPokemon.Hp -= player.Pokemons[player.Active].Atk
						opponent.Pokemons[opponent.Active] = opponentPokemon // Reassign modified Pokémon back to map
						if opponentPokemon.Hp <= 0 {
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
}

func loadPokemonData(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var pokemons struct {
		PlayerPokemons []PlayerPokemon
	}
	if err := json.Unmarshal(data, &pokemons); err != nil {
		return err
	}
	availablePokemons = pokemons.PlayerPokemons
	return nil
}

func formatPokemonList() string {
	var sb strings.Builder
	for _, p := range availablePokemons {
		sb.WriteString(fmt.Sprintf("%s (HP: %d, Attack: %d)", p.Name, p.Hp, p.Atk))

	}
	return sb.String()
}

func isInBattle(p string) bool {
	_, exists := inBattlePlayers[p]
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
