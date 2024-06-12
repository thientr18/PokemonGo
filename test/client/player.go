package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

var inBattle = make(map[*net.UDPAddr]bool)
var mu sync.Mutex

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "localhost:8080")
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}

	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter your username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)
		_, err = conn.Write([]byte("@join " + username))
		if err != nil {
			fmt.Println("Error joining chat:", err)
			return
		}

		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {

		}
		if string(buffer[:n]) == "duplicated_username" {
			fmt.Println("Duplicated username, choose other username!")
		} else {
			fmt.Println(string(buffer[:n]))
			break
		}
	}

	go receiveMessages(udpAddr, conn)

	for {
		mu.Lock()
		if !inBattle[udpAddr] {
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			_, err := conn.Write([]byte(text))
			if err != nil {
				fmt.Println("Error sending message:", err)
				mu.Unlock()
				return
			}
		}
		mu.Unlock()
	}
}

func receiveMessages(addr *net.UDPAddr, conn *net.UDPConn) {
	buffer := make([]byte, 1024)
	reader := bufio.NewReader(os.Stdin)

	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error receiving message:", err)
			return
		}
		response := string(buffer[:n])

		// Check if the user wants to quit
		if strings.Contains(response, "Goodbye") {
			fmt.Println(response)
			conn.Close()
			break
		}

		// Handle battle start
		if response == "Battle Started!" {
			handleBattle(addr, conn, reader)
			break
		}

		fmt.Println(response)
	}
}

func handleBattle(addr *net.UDPAddr, conn *net.UDPConn, reader *bufio.Reader) {
	mu.Lock()
	inBattle[addr] = true
	mu.Unlock()

	pokemonChoices := make([]string, 3)
	for i := 0; i < 3; i++ {
		fmt.Printf("Choose your Pokémon %d: ", i+1)
		pokemonChoices[i], _ = reader.ReadString('\n')
		pokemonChoices[i] = strings.TrimSpace(pokemonChoices[i])
	}

	chooseMsg := fmt.Sprintf("@pick %s %s %s", pokemonChoices[0], pokemonChoices[1], pokemonChoices[2])
	_, err := conn.Write([]byte(chooseMsg))
	if err != nil {
		fmt.Println("Error sending Pokémon choices:", err)
		return
	}

	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println("Error receiving response:", err)
		return
	}
	response := string(buffer[:n])
	if response == "@chosen" {
		fmt.Println("Pokémon chosen successfully! Battle begins!")
	} else {
		fmt.Println("Failed to choose Pokémon:", response)
		mu.Lock()
		inBattle[addr] = false
		mu.Unlock()
		return
	}

	for {
		fmt.Print("Enter command (@attack/@change [pokemon]): ")
		cmd, _ := reader.ReadString('\n')
		cmd = strings.TrimSpace(cmd)

		_, err := conn.Write([]byte(cmd))
		if err != nil {
			fmt.Println("Error sending command:", err)
			return
		}

		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error receiving response:", err)
			return
		}

		response := string(buffer[:n])
		fmt.Println("Response from server:", response)

		if strings.HasPrefix(response, "@win") {
			fmt.Println("You win!")
			break
		}
		if strings.HasPrefix(response, "@lose") {
			fmt.Println("You lose :<")
			break
		}
		// Check if the user wants to quit
		if strings.Contains(response, "Goodbye") {
			conn.Close()
			break
		}
	}

	mu.Lock()
	inBattle[addr] = false
	mu.Unlock()
}
