package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	HOST        = "localhost"
	PORT        = "8080"
	TYPE        = "udp"
	pokedexData = "src\\pokedex.json"
)

type (
	Pokemon struct {
		Id       string   `json:"ID"`
		Name     string   `json:"Name"`
		Link     string   `json:"URL"`
		PokeInfo PokeInfo `json:"Poke-Information"`
	}

	PokeInfo struct {
		Types       []string `json:"types"`
		Hp          int      `json:"HP"`
		Atk         int      `json:"ATK"`
		Def         int      `json:"DEF"`
		SpAtk       int      `json:"Sp.Atk"`
		SpDef       int      `json:"Sp.Def"`
		Speed       int      `json:"Speed"`
		TypeDefense TypeDef  `json:"Type-Defenses"`
	}
	TypeDef struct {
		Normal   float32
		Fire     float32
		Water    float32
		Electric float32
		Grass    float32
		Ice      float32
		Fighting float32
		Poison   float32
		Ground   float32
		Flying   float32
		Psychic  float32
		Bug      float32
		Rock     float32
		Ghost    float32
		Dragon   float32
		Dark     float32
		Steel    float32
		Fairy    float32
	}

	PlayerPokemon struct { // store pokemmon that a player holding
		Name        string `json:"Name"`
		ID          string
		Level       int
		Exp         int
		Types       []string `json:"types"`
		Hp          int      `json:"HP"`
		Atk         int      `json:"ATK"`
		Def         int      `json:"DEF"`
		SpAtk       int      `json:"Sp.Atk"`
		SpDef       int      `json:"Sp.Def"`
		Speed       int      `json:"Speed"`
		TypeDefense TypeDef  `json:"Type-Defenses"`
	}

	Player struct {
		Name                  string
		Addr                  *net.UDPAddr
		Pokemons              map[string]PlayerPokemon
		BattlePokemon         map[string]BattlePokemon
		battleRequestSends    map[string]string // store number of request that a player send: 'map[receivers]sender'
		battleRequestReceives map[string]string // store number of request that a player get: 'map[senders]receiver'
		Active                string
		battleID              int64
	}

	BattlePokemon struct {
		Name        string `json:"Name"`
		ID          string
		Level       int
		Exp         int
		Types       []string `json:"types"`
		Hp          int      `json:"HP"`
		Atk         int      `json:"ATK"`
		Def         int      `json:"DEF"`
		SpAtk       int      `json:"Sp.Atk"`
		SpDef       int      `json:"Sp.Def"`
		Speed       int      `json:"Speed"`
		TypeDefense TypeDef  `json:"Type-Defenses"`
	}

	Battle struct {
		battleID       int64
		Players        map[string]*Player
		ActivePokemons map[string]BattlePokemon // Store active Pokemons in the battle
		TurnOrder      []string
		Current        int
		Status         string // "waiting", "inviting", "active"
	}
)

var gameStates = make(map[int64]Battle)

var pokedex []Pokemon // pokedex

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
	if err := loadPokemonData("src\\playersPokemon.json"); err != nil {
		fmt.Println("Error loading players' pokemons data:", err)
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
					inBattleWith[senderName] = opponent
					inBattleWith[opponent] = senderName

					delete(players[opponent].battleRequestSends, senderName)
					delete(players[senderName].battleRequestReceives, opponent)

					var id = getNanoTime()

					gameStates[id] = Battle{
						battleID: id,
						Players:  make(map[string]*Player),
						Status:   "waiting",
					}

					gameStates[id].Players[senderName] = players[senderName]
					gameStates[id].Players[opponent] = players[opponent]

					players[senderName] = &Player{
						battleID: id,
					}
					players[opponent] = &Player{
						battleID: id,
					}

					sendMessage("You accepted a battle with player '"+opponent+"'", addr, conn)
					sendMessage("@accepted_battle", addr, conn)

					sendMessage("Your battle request with player '"+senderName+"' is accepted!", players[opponent].Addr, conn)
					sendMessage("@accepted_battle", players[opponent].Addr, conn)
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
			case "@pick":
				if len(strings.Split(message, " ")) != 4 {
					fmt.Println(len(parts))
					sendMessage("Invalid pokemons selection!", addr, conn)
					break
				}
				if _, exists := gameStates[players[senderName].battleID].Players[senderName]; exists &&
					gameStates[players[senderName].battleID].Status == "waiting" {

					for i := 1; i < 4; i++ {
						chosen := strings.Split(message, " ")[i] // choose: Pokemon picked
						if p, ok := gameStates[players[senderName].battleID].Players[senderName].Pokemons[chosen]; ok {
							gameStates[players[senderName].battleID].ActivePokemons[senderName+"_"+chosen] = BattlePokemon{Name: p.Name,
								ID:          p.ID,
								Level:       p.Level,
								Exp:         p.Exp,
								Hp:          p.Hp,
								Types:       p.Types,
								Atk:         p.Atk,
								Def:         p.Def,
								SpAtk:       p.SpAtk,
								SpDef:       p.SpDef,
								Speed:       p.Speed,
								TypeDefense: p.TypeDefense}
							fmt.Println("error")
						} else {
							sendMessage("Invalid pokemons selection!", addr, conn)
							break
						}
					}
					if len(gameStates[players[senderName].battleID].ActivePokemons) == 6 { // Both players have chosen their Pokémon
						gameStates[players[senderName].battleID] = Battle{
							Status: "active",
						}
						sendMessage("@pokemon_start_battle", addr, conn)
					} else {
						sendMessage("@pokemon_chosen", addr, conn)
					}
				} else {
					fmt.Println()
					sendMessage("No active game found", addr, conn)
				}
			case "@attack": // Ngân sửa
				var counState int = 0
				for _, game := range gameStates {
					if counState == 0 {
						opponentID := game.TurnOrder[(game.Current+1)%len(game.TurnOrder)] // len = 2, chưa hiẻu lắm nhưng nếu attack thì sẽ set cái TurnOrder cho đối thủ
						activePlayer := game.ActivePokemons[senderName+"_"+game.TurnOrder[game.Current]]
						activeOpponent := game.ActivePokemons[opponentID+"_"+game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]]
						if checkSpeed(activePlayer, activeOpponent) == " opponent" {
							game.TurnOrder[game.Current] = senderName
						} else {
							game.TurnOrder[game.Current] = opponentID
						}
					}

					if _, ok := game.Players[senderName]; ok {
						if game.TurnOrder[game.Current] != senderName { // turn based
							sendMessage("Not your turn!", addr, conn)
						}

						opponentID := game.TurnOrder[(game.Current+1)%len(game.TurnOrder)] // len = 2, chưa hiẻu lắm nhưng nếu attack thì sẽ set cái TurnOrder cho đối thủ
						if _, ok := game.Players[opponentID]; ok {

							activePlayer := game.ActivePokemons[senderName+"_"+game.TurnOrder[game.Current]]

							activeOpponent := game.ActivePokemons[opponentID+"_"+game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]]

							activeOpponent.Hp -= int(getDmgNumber(activePlayer, activeOpponent))

							game.ActivePokemons[opponentID+"_"+game.TurnOrder[(game.Current+1)%len(game.TurnOrder)]] = activeOpponent

							if activeOpponent.Hp <= 0 {
								delete(game.ActivePokemons, opponentID+"_"+game.TurnOrder[(game.Current+1)%len(game.TurnOrder)])

							}

							if _, oke := game.ActivePokemons[opponentID]; oke {
								conn.WriteToUDP([]byte("@attacke "+senderName), addr)
								game.Current = (game.Current + 1) % len(game.TurnOrder)
							} else {
								conn.WriteToUDP([]byte("@win"+senderName), addr)
							}

						}

					}
					counState++
				}
			case "@change":
				for _, game := range gameStates {
					if player, ok := game.Players[senderName]; ok {
						if game.TurnOrder[game.Current] != senderName {
							sendMessage("Not your turn", addr, conn)
							break
						}
						if len(parts) == 2 {
							newActive := parts[1]
							if _, ok := player.Pokemons[newActive]; ok {
								game.TurnOrder[game.Current] = newActive
								sendMessage("@changed", addr, conn)
								game.Current = (game.Current + 1) % len(game.TurnOrder)
							} else {
								sendMessage("Invalid Pokemon", addr, conn)
							}
						}
					}
				}

			case "@surrender":
				battle := gameStates[players[senderName].battleID]

				surrenderer, exists := battle.Players[senderName]
				if !exists {
					fmt.Println("Error: Surrenderer not found in the battle.")
					return
				}

				var opponent *Player
				for name, player := range battle.Players {
					if name != senderName {
						opponent = player
						break
					}
				}

				if opponent == nil {
					fmt.Println("Error: No opponent found.")
					return
				}

				winner := opponent

				battle.Status = "finished"

				sendMessage("You surrendered. "+winner.Name+" wins!", surrenderer.Addr, conn)
				sendMessage("Your opponent surrendered. You win!", opponent.Addr, conn)

				totalExp := 0
				for _, pokemon := range surrenderer.Pokemons {
					totalExp += pokemon.Exp //total exp of loser pokemon after picked
				} //fix this??? but it is wrong logic

				ExpToDistribute := totalExp / 3 //distribute three pokemon on loser team to winning team
				//correct logic

				for _, pokemon := range winner.Pokemons {
					pokemon.Exp += ExpToDistribute //total exp of winnder
				} //correct logic

				fmt.Printf("%s's Pokemon after battle: \n", winner.Name)
				for _, pokemon := range winner.Pokemons {
					fmt.Printf("%d\n", pokemon.Exp)
				}

				delete(inBattleWith, senderName)
				delete(inBattleWith, opponent.Name)

				fmt.Printf("Player '%s' surrendered. Player '%s' wins!\n", senderName, winner.Name)

				//from here this code belongs to the method
				// game, exists := gameState.Battles[senderName]
				// if !exists {
				// 	sendMessage("No active game found", addr, conn)
				// 	break
				// }
				// game.Surrender(senderName)

			case "@y":
				sendMessage("@pokemon_list_pick"+formatPokemonList(), addr, conn)
			case "@n":
				sendMessage("@pokemon_pick", addr, conn)

			default:
				sendMessage("Invalid command 2", addr, conn)
			}
		}
	} else {
		fmt.Println("Invalid command 3")
		sendMessage("Invalid command 3", addr, conn)
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
	return json.Unmarshal(data, &pokedex) // gán data vào pokedex
}
func getDmgNumber(pAtk BattlePokemon, pRecive BattlePokemon) float32 {
	var dmg float32
	var types = make(map[string]float32)

	types["Normal"] = pRecive.TypeDefense.Normal
	types["Fire"] = pRecive.TypeDefense.Fire
	types["Water"] = pRecive.TypeDefense.Water
	types["Electric"] = pRecive.TypeDefense.Electric
	types["Grass"] = pRecive.TypeDefense.Grass
	types["Ice"] = pRecive.TypeDefense.Ice
	types["Fighting"] = pRecive.TypeDefense.Fighting
	types["Poison"] = pRecive.TypeDefense.Poison
	types["Ground"] = pRecive.TypeDefense.Ground
	types["Flying"] = pRecive.TypeDefense.Flying
	types["Psychic"] = pRecive.TypeDefense.Psychic
	types["Bug"] = pRecive.TypeDefense.Bug
	types["Rock"] = pRecive.TypeDefense.Rock
	types["Ghost"] = pRecive.TypeDefense.Ghost
	types["Dragon"] = pRecive.TypeDefense.Dragon
	types["Dark"] = pRecive.TypeDefense.Dark
	types["Steel"] = pRecive.TypeDefense.Steel
	types["Fairy"] = pRecive.TypeDefense.Fairy

	rand.Seed(time.Now().UnixNano())
	var choseAtk = rand.Intn(1-0+1) + 0
	if choseAtk == 0 {
		dmg = float32(pAtk.Atk) - float32(pRecive.Def)
	} else {
		var typeDefense float32 = 0.0
		for _, pAtkTypes := range pAtk.Types {
			for typeDef, def := range types {
				if typeDef == pAtkTypes {
					if typeDefense < def {
						typeDefense = def
					}
				}
			}
		}
		dmg = float32(pAtk.SpAtk)*typeDefense - float32(pRecive.SpDef)
	}
	return dmg

}
func checkSpeed(pAtk BattlePokemon, pRecive BattlePokemon) string {
	if pAtk.Speed > pRecive.Speed {
		return "pLayer"
	} else if pAtk.Speed > pRecive.Speed {
		return "opponent"
	} else {
		var chosePokemon = rand.Intn(1-0+1) + 0
		if chosePokemon == 0 {
			return "PLayer"
		} else {
			return "opponent"
		}
	}
}

func getNanoTime() int64 {
	return time.Now().UnixNano()
}

// func (battle *Battle) Surrender(senderName string) {
// 	surrenderer, exists := battle.Players[senderName]
// 	if !exists {
// 		fmt.Println("Error: Surrenderer not found in the battle.")
// 		return
// 	}

// 	var opponent *Player
// 	for name, player := range battle.Players {
// 		if name != senderName {
// 			opponent = player
// 			break
// 		}
// 	}

// 	if opponent == nil {
// 		fmt.Println("Error: No opponent found.")
// 		return
// 	}

// 	winner := opponent

// 	battle.Status = "finished"

// 	sendMessage("You surrendered. "+winner.Name+" wins!", surrenderer.Addr, conn)
// 	sendMessage("Your opponent surrendered. You win!", opponent.Addr, conn)

// 	totalExp := 0
// 	for _, pokemon := range surrenderer.Pokemons {
// 		totalExp += pokemon.Exp //total exp of loser pokemon after picked
// 	} //fix this??? but it is wrong logic

// 	ExpToDistribute := totalExp / 3 //distribute three pokemon on loser team to winning team
// 	//correct logic

// 	for _, pokemon := range winner.Pokemons {
// 		pokemon.Exp += ExpToDistribute //total exp of winnder
// 	} //correct logic

// 	fmt.Printf("%s's Pokemon after battle: \n", winner.Pokemons)
// 	for _, pokemon := range winner.Pokemons {
// 		fmt.Printf("%s\n", pokemon)
// 	}

// 	delete(inBattleWith, senderName)
// 	delete(inBattleWith, opponent.Name)

// 	fmt.Printf("Player '%s' surrendered. Player '%s' wins!\n", senderName, winner.Name)
// }
