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
	HOST               = "localhost"
	PORT               = "8080"
	TYPE               = "udp"
	pokedexData        = "src\\pokedex.json"
	playerpokemonsData = "src\\playersPokemon.json"
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
		Normal   float32 `json:"Normal"`
		Fire     float32 `json:"Fire"`
		Water    float32 `json:"Water"`
		Electric float32 `json:"Electric"`
		Grass    float32 `json:"Grass"`
		Ice      float32 `json:"Ice"`
		Fighting float32 `json:"Fighting"`
		Poison   float32 `json:"Poison"`
		Ground   float32 `json:"Ground"`
		Flying   float32 `json:"Flying"`
		Psychic  float32 `json:"Psychic"`
		Bug      float32 `json:"Bug"`
		Rock     float32 `json:"Rock"`
		Ghost    float32 `json:"Ghost"`
		Dragon   float32 `json:"Dragon"`
		Dark     float32 `json:"Dark"`
		Steel    float32 `json:"Steel"`
		Fairy    float32 `json:"Fairy"`
	}

	Player struct {
		Name                  string `json:"PlayerName"`
		Addr                  *net.UDPAddr
		Pokemons              map[string]PlayerPokemon // string là pokemon ID
		BattlePokemon         map[string]BattlePokemon
		battleRequestSends    map[string]string // store number of request that a player send: 'map[receivers]sender'
		battleRequestReceives map[string]string // store number of request that a player get: 'map[senders]receiver'
		Active                string
		battleID              int64
	}

	PlayerPokemon struct { // store pokemmon that a player holding
		Owner          string           `json:"PlayerName"`
		PlayerPokeInfo []PlayerPokeInfo `json:"Pokemons"`
	}
	PlayerPokeInfo struct { // store pokemmon that a player holding
		ID          string   `json:"ID"`
		Name        string   `json:"Name"`
		Level       int      `json:"Level"`
		Exp         int      `json:"Exp"`
		Types       []string `json:"types"`
		Hp          int      `json:"HP"`
		Atk         int      `json:"ATK"`
		Def         int      `json:"DEF"`
		SpAtk       int      `json:"Sp.Atk"`
		SpDef       int      `json:"Sp.Def"`
		Speed       int      `json:"Speed"`
		TypeDefense TypeDef  `json:"Type-Defenses"`
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
		ActivePokemons map[string]*BattlePokemon // Store active Pokemons in the battle
		BeatingPokemon map[string]*BattlePokemon
		CurrentTurn    string
		Status         string // "waiting", "inviting", "active"
		PokemonCounter map[string]int
	}
)

var pokedex []Pokemon // pokedex

var playersPokemons []PlayerPokemon // player's Pokemons

var players = make(map[string]*Player) // list of player online

var inBattleWith = make(map[string]string) // check player is in battle or not

var gameStates = make(map[int64]*Battle) // battles

func loadPlayerPokemon(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &playersPokemons) // gán data vào pokedex
}

func main() {
	// Load the pokedex data from the JSON file
	err := loadPokedex(pokedexData)
	if err != nil {
		fmt.Println("Error loading pokedex data:", err)
	}

	err = loadPlayerPokemon(playerpokemonsData)
	if err != nil {
		fmt.Println("Error loading pokedex data:", err)
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
						battleID:              0,
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

				if opponent == senderName {
					sendMessage("Invalid command", addr, conn)
					break
				}

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
				nextPart := strings.Split(temp, " ")
				opponent := nextPart[0]

				if players[senderName].battleRequestReceives[opponent] == senderName &&
					players[opponent].battleRequestSends[senderName] == opponent {

					inBattleWith[senderName] = opponent
					inBattleWith[opponent] = senderName

					delete(players[opponent].battleRequestSends, senderName)
					delete(players[senderName].battleRequestReceives, opponent)

					var id = getNanoTime()

					gameStates[id] = &Battle{
						battleID:       id,
						Players:        make(map[string]*Player),
						ActivePokemons: make(map[string]*BattlePokemon),
						BeatingPokemon: make(map[string]*BattlePokemon),
						CurrentTurn:    players[senderName].Name,
						Status:         "waiting",
						PokemonCounter: make(map[string]int),
					}

					gameStates[id].Players[senderName] = players[senderName]
					gameStates[id].Players[opponent] = players[opponent]

					players[senderName] = &Player{
						battleID:              id,
						Name:                  senderName,
						Addr:                  addr,
						battleRequestSends:    make(map[string]string),
						battleRequestReceives: make(map[string]string),
					}
					players[opponent] = &Player{
						battleID:              id,
						Name:                  opponent,
						Addr:                  players[opponent].Addr,
						battleRequestSends:    make(map[string]string),
						battleRequestReceives: make(map[string]string),
					}

					players[opponent].battleID = id
					players[opponent].battleID = id

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
				// Find player Pokemons by player's name
				playerPokemons := findPlayerPokemonByPlayer(senderName)
				fmt.Printf("Pokémons of player %s:\n", senderName)
				for _, pokemon := range playerPokemons {
					str := fmt.Sprintf("Pokemon ID: %s, Name: %s, Level: %d, HP: %d\n", pokemon.ID, pokemon.Name, pokemon.Level, pokemon.Hp)
					sendMessage("@list_pokemon_only"+str, addr, conn)
				}

			case "@pokedex":
				parts = strings.Split(message, " ")
				sendMessage("@pokedex"+pokedexScanner(parts[1]), addr, conn)
			default:
				sendMessage("Invalid command, not in a battle!", addr, conn)
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
				parts = strings.Split(message, " ")
				if len(parts) != 4 {
					fmt.Println(len(parts))
					sendMessage("Invalid pokemons selection!", addr, conn)
					break
				} else if parts[1] == parts[2] || parts[1] == parts[3] || parts[2] == parts[3] {
					fmt.Println(len(parts))
					sendMessage("Invalid pokemons selection!", addr, conn)
					break
				}
				gameStates[players[senderName].battleID].PokemonCounter[senderName] = 0
				if _, exists := gameStates[players[senderName].battleID].Players[senderName]; exists &&
					gameStates[players[senderName].battleID].Status == "waiting" {

					for i := 1; i < 4; i++ {
						chosen := parts[i] // choose: Pokemon picked
						p := findPlayerPokemonByPokeID(senderName, chosen)
						if p != nil {
							gameStates[players[senderName].battleID].ActivePokemons[senderName+"_"+p.ID] = &BattlePokemon{
								Name:        p.Name,
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
							gameStates[players[senderName].battleID].PokemonCounter[senderName] += 1
						} else {
							sendMessage("Invalid pokemons selection!", addr, conn)
							break
						}
					}

					if len(gameStates[players[senderName].battleID].ActivePokemons) == 6 { // Both players have chosen their Pokémon
						gameStates[players[senderName].battleID].Status = "active"
						sendMessage("@pokemon_start_battle", addr, conn)
						sendMessage("@pokemon_start_battle", players[inBattleWith[senderName]].Addr, conn)

						id := players[senderName].battleID
						var firstPokemonOpponent = gameStates[id].ActivePokemons[inBattleWith[senderName]+"_"+findPlayerPokemonByPokeID(inBattleWith[senderName], parts[1]).ID]
						var firstPokemonSenderName = gameStates[id].ActivePokemons[senderName+"_"+findPlayerPokemonByPokeID(senderName, parts[1]).ID]

						gameStates[id].BeatingPokemon[senderName] = firstPokemonSenderName             // set pokemon đang đấm nhau hiện tại
						gameStates[id].BeatingPokemon[inBattleWith[senderName]] = firstPokemonOpponent // set pokemon đang đấm nhau hiện tại

						if firstPokemonOpponent.Speed > firstPokemonSenderName.Speed {
							gameStates[id].CurrentTurn = inBattleWith[senderName]
						} else if firstPokemonOpponent.Speed < firstPokemonSenderName.Speed {
							gameStates[id].CurrentTurn = senderName
						}

						if gameStates[id].CurrentTurn == senderName {
							sendMessage("You attack first!", addr, conn)
							msg := fmt.Sprintf("Active Pokemon: %s (HP: %d)", gameStates[id].BeatingPokemon[inBattleWith[senderName]].Name, gameStates[id].BeatingPokemon[inBattleWith[senderName]].Hp)
							sendMessage(msg, players[inBattleWith[senderName]].Addr, conn)
							sendMessage("Opponent will attack first!", players[inBattleWith[senderName]].Addr, conn)
							msg = fmt.Sprintf("Active Pokemon: %s (HP: %d)", gameStates[id].BeatingPokemon[senderName].Name, gameStates[id].BeatingPokemon[senderName].Hp)
							sendMessage(msg, players[senderName].Addr, conn)
						} else {
							sendMessage("You attack first!", players[inBattleWith[senderName]].Addr, conn)
							sendMessage("Opponent will attack first!", addr, conn)
						}
					} else {
						sendMessage("@pokemon_picked", addr, conn)
					}
				} else {
					fmt.Println()
					sendMessage("No active game found", addr, conn)
				}
			case "@attack":
				id := players[senderName].battleID
				if gameStates[id].CurrentTurn != senderName {
					sendMessage("Not your turn!", addr, conn)
					break
				}
				opponent := inBattleWith[senderName]
				currPoke := findPlayerPokemonByPokeID(opponent, gameStates[id].BeatingPokemon[opponent].ID)

				dmg := int(getDmgNumber(gameStates[id].BeatingPokemon[senderName], gameStates[id].BeatingPokemon[opponent]))
				// gameStates[id].BeatingPokemon[opponent].Hp -= dmg
				gameStates[id].ActivePokemons[opponent+"_"+currPoke.ID].Hp -= dmg

				msg := fmt.Sprintf("%s hits: %d damages!", gameStates[id].BeatingPokemon[senderName].Name, dmg)
				sendMessage(msg, addr, conn)
				msg = fmt.Sprintf("%s hited: %d damages!", gameStates[id].BeatingPokemon[opponent].Name, dmg)
				sendMessage(msg, players[opponent].Addr, conn)

				msg = fmt.Sprintf("Active Pokemon: %s (HP: %d)", gameStates[id].BeatingPokemon[opponent].Name, gameStates[id].BeatingPokemon[opponent].Hp)
				sendMessage(msg, players[opponent].Addr, conn)
				msg = fmt.Sprintf("Active Pokemon: %s (HP: %d)", gameStates[id].BeatingPokemon[senderName].Name, gameStates[id].BeatingPokemon[senderName].Hp)
				sendMessage(msg, addr, conn)

				gameStates[id].CurrentTurn = opponent

				if gameStates[id].BeatingPokemon[opponent].Hp <= 0 {
					sendMessage("Your pokemon died, change the order!", players[opponent].Addr, conn)
					sendMessage("@pokemon_died", players[opponent].Addr, conn)
					delete(gameStates[id].ActivePokemons, opponent+"_"+gameStates[id].BeatingPokemon[opponent].ID)
					delete(gameStates[id].BeatingPokemon, opponent)
					gameStates[id].PokemonCounter[opponent] -= 1
				}

				if gameStates[id].PokemonCounter[opponent] > 0 {
					sendMessage("@opponent_attacked", players[opponent].Addr, conn)
					sendMessage("@you_acttacked", addr, conn)
				} else {
					sendMessage("@win", addr, conn)
					sendMessage("@lose", players[opponent].Addr, conn)
					delete(inBattleWith, opponent)
					delete(inBattleWith, senderName)
				}
			case "@change":
				parts := strings.Split(message, " ")
				if len(parts) < 2 {
					sendMessage("Invalid pokemon name", addr, conn)
					break
				}
				id := players[senderName].battleID
				if gameStates[id].CurrentTurn != senderName {
					sendMessage("Not your turn!", addr, conn)
					break
				}

				opponent := inBattleWith[senderName]
				pokemonKey := senderName + "_" + parts[1]

				if activePokemon, exists := gameStates[id].ActivePokemons[pokemonKey]; exists {
					gameStates[id].BeatingPokemon[senderName] = activePokemon
					sendMessage("@changed", addr, conn)
					gameStates[id].CurrentTurn = opponent
				} else {
					sendMessage("Invalid Pokemon", addr, conn)
				}
			case "@y":
				playerPokemons := findPlayerPokemonByPlayer(senderName)
				fmt.Printf("Pokémons of player %s:\n", senderName)
				var str string
				for _, pokemon := range playerPokemons {
					str += fmt.Sprintf("Pokemon ID: %s, Name: %s, Level: %d, HP: %d\n", pokemon.ID, pokemon.Name, pokemon.Level, pokemon.Hp)

				}
				sendMessage("@list_then_pick_pokemon"+str, addr, conn)
			case "@n":
				sendMessage("@pick_only", addr, conn)
			default:
				sendMessage("Invalid command 2", addr, conn)
			}
		}
	} else {
		sendMessage("Invalid command in a battle", addr, conn)
	}
}

func loadPokedex(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &pokedex) // gán data vào pokedex
}

// load data in playersPokemon.json
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
	playersPokemons = pokemons.Pokemons
	return nil
}

// find pokemon in a pokedex
func findPokemonByNameOrID(name string) *Pokemon {
	for _, p := range pokedex {
		if p.Name == name || p.Id == name {
			return &p
		}
	}
	return nil
}

func findPlayerPokemonByPlayer(playerName string) []PlayerPokeInfo {
	var pokemon []PlayerPokeInfo
	for _, p := range playersPokemons {
		if p.Owner == playerName {
			pokemon = p.PlayerPokeInfo
		}
	}
	return pokemon
}

func findPlayerPokemonByPokeID(playerName string, idPoke string) *PlayerPokeInfo {
	for _, p := range playersPokemons {
		if p.Owner == playerName {
			for _, po := range p.PlayerPokeInfo {
				if po.ID == idPoke {
					return &po
				}
			}
		}
	}
	return nil
}

func pokedexScanner(pokeName string) string {
	pokemon := findPokemonByNameOrID(pokeName)
	if pokemon == nil {
		return fmt.Sprintf("Pokémon with name %s not found", pokeName)
	}
	return fmt.Sprintf("ID: %s\nName: %s\nTypes: [%s]\nBase Stats: HP: %d, ATK: %d, DEF: %d, Sp.Atk: %d, Sp.Def: %d, Speed: %d",
		pokemon.Id, pokemon.Name, pokemon.PokeInfo.Types[:], pokemon.PokeInfo.Hp, pokemon.PokeInfo.Atk, pokemon.PokeInfo.Def,
		pokemon.PokeInfo.SpAtk, pokemon.PokeInfo.SpDef, pokemon.PokeInfo.Speed)
}

func isInBattle(p string) bool {
	_, exists := inBattleWith[p]
	if !exists {
		return false
	} else {
		return true
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

func sendMessage(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	_, err := conn.WriteToUDP([]byte(message), addr)
	if err != nil {
		fmt.Println("Error sending message:", err)
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

func getDmgNumber(pAtk *BattlePokemon, pRecive *BattlePokemon) int {
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
	choseAtk := rand.Intn(2)
	if choseAtk == 0 {
		dmg = float32(pAtk.Atk) - float32(pRecive.Def)
		if dmg < 0 {
			dmg = 0
		}
		return int(dmg)
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
		if dmg < 0 {
			dmg = 0
		}
		return int(dmg)
	}
}
func checkSpeed(pAtk *BattlePokemon, pRecive *BattlePokemon) string {
	if pAtk.Speed > pRecive.Speed {
		return "player"
	} else if pAtk.Speed > pRecive.Speed {
		return "opponent"
	} else {
		return "player"
	}
}

func getNanoTime() int64 {
	return time.Now().UnixNano()
}
