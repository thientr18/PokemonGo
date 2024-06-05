package main

import (
	"fmt"
	"net"
	"strings"
)

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
		case "@attack":
			if players[senderName].isInBattle() {
				attack()
			}
		case "@change":
			if players[senderName].isInBattle() {
				temp := parts[1]
				nextPart := strings.SplitN(temp, " ", 2)
				pokemonID := nextPart[0]
				changePokemon(pokemonID)
			}
		default:
			sendMessage("Invalid command", senderName, conn)
		}
	} else {
		sendMessage("Invalid command format", getPlayernameByAddr(addr), conn)
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

func sendMessage(message, username string, conn *net.UDPConn) {
	player := players[username]
	_, err := conn.WriteToUDP([]byte(message), player.Addr)
	if err != nil {
		fmt.Println("Error sending private message:", err)
	}
}
