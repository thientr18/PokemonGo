package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

var inBattle bool
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
		fmt.Println("Error receiving response:", err)
		return
	}
	if string(buffer[:n]) == "duplicated-username" {
		fmt.Println("Duplicated username, choose another username!")
		return
	} else {
		fmt.Println(string(buffer[:n]))
	}

	go receiveMessages(conn)

	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		mu.Lock()
		if inBattle {
			handleBattleCommand(conn, text)
		} else {
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

func receiveMessages(conn *net.UDPConn) {
	buffer := make([]byte, 1024)

	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error receiving message:", err)
			return
		}
		response := string(buffer[:n])
		fmt.Println(response)

		// Check if the user wants to quit
		if strings.Split(response, ", ")[0] == "Goodbye" {
			os.Exit(0)
			break
		}

		// Handle battle start
		if response == "Battle Started!" {
			mu.Lock()
			inBattle = true
			mu.Unlock()
			fmt.Println("Battle started!")
			handleBattleSetup(conn)
		}
	}
}

func handleBattleSetup(conn *net.UDPConn) {
	reader := bufio.NewReader(os.Stdin)
	buffer := make([]byte, 1024)

	for {
		fmt.Println("Do you want to see the list of available Pokémon? (yes/no)")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "yes" {
			_, err := conn.Write([]byte("@list"))
			if err != nil {
				fmt.Println("Error requesting Pokémon list:", err)
				return
			}
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Println("Error receiving Pokémon list:", err)
				return
			}
			fmt.Println("Available Pokémon:", strings.TrimPrefix(string(buffer[:n]), "@list"))
			break
		} else if choice == "no" {
			break
		} else {
			fmt.Println("Invalid choice, please enter 'yes' or 'no'")
		}
	}

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
		inBattle = false
		mu.Unlock()
	}
}

func handleBattleCommand(conn *net.UDPConn, text string) {
	_, err := conn.Write([]byte(text))
	if err != nil {
		fmt.Println("Error sending command:", err)
		return
	}

	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println("Error receiving response:", err)
		return
	}

	response := string(buffer[:n])
	fmt.Println("Response from server:", response)

	if strings.HasPrefix(response, "@win") {
		fmt.Println("You win!")
		mu.Lock()
		inBattle = false
		mu.Unlock()
	}
	if strings.HasPrefix(response, "@lose") {
		fmt.Println("You lose :<")
		mu.Lock()
		inBattle = false
		mu.Unlock()
	}
}
