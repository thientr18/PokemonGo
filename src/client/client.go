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
	addr, err := net.ResolveUDPAddr("udp", "localhost:8080")
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
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

	go receiveMessages(addr, conn)

	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		_, err := conn.Write([]byte(text))
		if err != nil {
			fmt.Println("Error sending message:", err)
			mu.Unlock()
			return
		}
	}
}

func receiveMessages(addr *net.UDPAddr, conn *net.UDPConn) {
	buffer := make([]byte, 1024)

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
			os.Exit(0)
			break
		}

		if strings.HasPrefix(response, "@pokemon_list") {
			fmt.Println("Your pokemons list: ")
			fmt.Println(strings.TrimPrefix(response, "@pokemon_list"))
			continue
		}

		// Handle battle start
		if strings.Contains(response, "@accepted_battle") {
			fmt.Println("Battle Started!")
			fmt.Println("See your pokemon list before selecting pokemons?\n[@y]: yes\n[@n]: no")
			continue
		}

		if strings.HasPrefix(response, "@pokemon_list_pick") {
			fmt.Println("Your pokemons list: ")
			fmt.Println(strings.TrimPrefix(response, "@pokemon_list"))
			fmt.Println("Choose your three pokemons for battle!\n(@pick pokemon1_ID pokemon2_ID pokemon3_ID)")
			continue
		}

		if strings.Contains(response, "@pokemon_pick") {
			fmt.Println("Choose your three pokemons for battle!\n(@pick pokemon1_ID pokemon2_ID pokemon3_ID)")
			continue
		}

		if strings.Contains(response, "@pokemon_piked") {
			fmt.Println("Pokémon picked successfully!\nWaiting your opponent...")
			continue
		}

		if strings.Contains(response, "@pokemon_start_battle") {
			fmt.Println("The battle begins! Faster pokemon moves first!")
			continue
		}

		if strings.Contains(response, "@changed") {
			fmt.Println("Pokémon picked successfully! Now is the oppenonent's turn!")
			continue
		}

		if strings.Contains(response, "@win") {
			fmt.Println("You win!")
			continue
		}

		if strings.Contains(response, "@lose") {
			fmt.Println("You lose :<")
			continue
		}

		fmt.Println(response)
	}
}

func checkInBattle(addr *net.UDPAddr) bool {
	_, exists := inBattle[addr]
	if !exists {
		return false
	} else {
		return true
	}
}
