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
		battleRequestSends    map[string]string // store number of request that a player send: 'map[receivers]sender'
		battleRequestReceives map[string]string // store number of request that a player get: 'map[senders]receiver'
		Active                string
	}

	BattlePokemon struct {
		Name string
		HP   int
		ATK  int
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

var availablePokemons []PlayerPokemon // store pokemons of player

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
					break
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
					sendMessage(privateMessage, players[receiver].Addr, conn)
				}
			case "@battle":
				if len(parts) < 2 {
					sendMessage("Invalid command", addr, conn)
					break
				}

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

				gameState.Battles[senderName] = &Battle{
					TurnOrder:      []string{senderName, opponent},
					Current:        0,
					Players:        map[string]*Player{},
					ActivePokemons: map[string]BattlePokemon{},
					Status:         "inviting",
				}

				battleRequestMessage := "Player '" + senderName + "' requests you a pokemon battle!"
				sendMessage(battleRequestMessage, players[opponent].Addr, conn)
			case "@accept":
				if len(parts) < 2 {
					sendMessage("Invalid command", addr, conn)
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

					inBattleWith[senderName] = opponent
					inBattleWith[opponent] = senderName

					delete(players[opponent].battleRequestSends, senderName)
					delete(players[senderName].battleRequestReceives, opponent)

					game := gameState.Battles[opponent]
					game.Players[senderName] = gameState.Players[senderName]
					game.Status = "waiting"
				} else {
					sendMessage("Invalid acception! (WRONG opppent name or NOT RECEIVES battle request from this opponent)", addr, conn)
				}
			case "@deny":
				if len(parts) < 2 {
					sendMessage("Invalid command", addr, conn)
					break
				}

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
			case "@list":
				sendMessage("@pokemonlist"+formatPokemonList(), addr, conn)
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
				if len(parts) < 2 {
					sendMessage("Invalid command", addr, conn)
					break
				}

				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				receiver := nextPart[0]
				if receiver != inBattleWith[senderName] {
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
				if len(parts) < 2 {
					sendMessage("Invalid command", addr, conn)
					break
				}

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
			case "@choose":
				if len(parts) != 4 {
					sendMessage("Invalid choose command", addr, conn)
					break
				}
				if game, exists := gameState.Battles[senderName]; exists && game.Status == "waiting" {
					for i := 1; i < 4; i++ {
						chosen := parts[i]
						if p, ok := gameState.Players[senderName].Pokemons[chosen]; ok {
							game.ActivePokemons[senderName+"_"+chosen] = BattlePokemon{Name: p.Name, HP: p.Hp, ATK: p.Atk} // Thêm ID
						} else {
							sendMessage("Invalid Pokémon selection", addr, conn)
							return
						}
					}
					if len(game.ActivePokemons) == 6 { // Both players have chosen their Pokémon
						game.Status = "active"
						conn.WriteToUDP([]byte("START|"+senderName), addr)
					} else {
						conn.WriteToUDP([]byte("CHOSEN|"+senderName), addr)
					}
				} else {
					conn.WriteToUDP([]byte("ERROR|No active game found"), addr)
				}
			case "@attack":
				for _, game := range gameState.Battles {
					if _, ok := game.Players[senderName]; ok {
						if game.TurnOrder[game.Current] != senderName {
							conn.WriteToUDP([]byte("ERROR|Not your turn"), addr)
							return
						}
						opponentID := game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]
						if _, ok := game.Players[opponentID]; ok {
							activePlayer := game.ActivePokemons[senderName+"_"+game.TurnOrder[game.Current]]
							activeOpponent := game.ActivePokemons[opponentID+"_"+game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]]
							activeOpponent.HP -= activePlayer.ATK
							game.ActivePokemons[opponentID+"_"+game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]] = activeOpponent
							if activeOpponent.HP <= 0 {
								conn.WriteToUDP([]byte("WIN|"+senderName), addr)
							} else {
								conn.WriteToUDP([]byte("ATTACKED|"+senderName), addr)
								game.Current = (game.Current + 1) % len(game.TurnOrder) // Change turn after attack
							}
						}
					}
				}
			case "@change":
				for _, game := range gameState.Battles {
					if player, ok := game.Players[senderName]; ok {
						if game.TurnOrder[game.Current] != senderName {
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
			case "@list":
				sendMessage("@pokemonlist"+formatPokemonList(), addr, conn)
			default:
				sendMessage("Invalid command", addr, conn)
			}
		}
	} else {
		sendMessage("Invalid command", addr, conn)
	}
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
