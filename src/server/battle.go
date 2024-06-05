package main

import (
	"fmt"
	"net"
)

var (
	numCurrentUser int
	numMaxUser     int
)

func attack() {
	panic("unimplemented")
}

func changePokemon(pokemonID string) {
	panic("unimplemented")
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
