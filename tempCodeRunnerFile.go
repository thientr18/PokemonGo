package main

import (
	"fmt"
)

type Player struct {
	battleRequestSends    map[string]string // store number of requests that a player sends: 'map[receivers]player'
	battleRequestReceives map[string]string // store number of requests that a player gets: 'map[senders]player'
}

var players = make(map[string]*Player)

func main() {
	senderName := "anh"
	opponent := "thien"

	// Initialize the players if they do not exist
	if players[senderName] == nil {
		players[senderName] = &Player{
			battleRequestSends:    make(map[string]string),
			battleRequestReceives: make(map[string]string),
		}
	}

	if players[opponent] == nil {
		players[opponent] = &Player{
			battleRequestSends:    make(map[string]string),
			battleRequestReceives: make(map[string]string),
		}
	}

	players[senderName].battleRequestSends[opponent] = senderName
	players[opponent].battleRequestReceives[senderName] = opponent
	fmt.Println(players[senderName].battleRequestSends[opponent])
	fmt.Println(players[opponent].battleRequestReceives[senderName])

	players["hehe"].battleRequestSends["hihi"] = "hehe"
	players["hihi"].battleRequestReceives["hehe"] = "hehe"
	fmt.Println(players["hehe"].battleRequestSends["hihi"])

}
