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

var canNotAttack = make(map[*net.UDPAddr]bool)

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
		if !strings.Contains(text, "@change") && canNotAttack[addr] == true {
			fmt.Println("Please change new pokemon first!")
			continue
		}
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

		if strings.Contains(response, "@list_then_pick_pokemon") {
			fmt.Println("Your pokemons list: ")
			fmt.Println(strings.TrimPrefix(response, "@list_then_pick_pokemon"))
			fmt.Println("Choose your three pokemons for battle!\n(@pick pokemon1_ID pokemon2_ID pokemon3_ID)")
			continue
		}

		if strings.Contains(response, "@list_pokemon_only") {
			fmt.Println("Your pokemons list: ")
			fmt.Println(strings.TrimPrefix(response, "@list_pokemon_only"))
			continue
		}

		// Handle battle start
		if strings.Contains(response, "@accepted_battle") {
			fmt.Println("Battle Started!")
			fmt.Println("See your pokemon list before selecting pokemons?\n[@y]: yes\n[@n]: no")
			continue
		}

		if strings.Contains(response, "@pick_only") {
			fmt.Println("Choose your three pokemons for battle!\n(@pick pokemon1_ID pokemon2_ID pokemon3_ID)")
			continue
		}

		if strings.Contains(response, "@pokemon_picked") {
			fmt.Println("Pokémon picked successfully!\nWaiting your opponent...")
			continue
		}

		if strings.Contains(response, "@pokemon_start_battle") {
			fmt.Println("The battle begins! Faster pokemon moves first!")
			continue
		}

		if strings.Contains(response, "@changed") {
			fmt.Println("Pokémon changed successfully! Now is the oppenonent's turn!")
			delete(canNotAttack, addr)
			continue
		}

		if strings.Contains(response, "@win") {
			fmt.Println("You win!")
			delete(canNotAttack, addr)
			continue
		}

		if strings.Contains(response, "@lose") {
			fmt.Println("You lose :<")
			delete(canNotAttack, addr)
			continue
		}

		if strings.Contains(response, "@pokedex") {
			fmt.Println(strings.TrimPrefix(response, "@pokedex"))
			continue
		}

		if strings.Contains(response, "@opponent_attacked") {
			fmt.Println("Your turn!")
			continue
		}

		if strings.Contains(response, "@you_acttacked") {
			fmt.Println("Opponent's turn!")
			continue
		}

		if strings.Contains(response, "@pokemon_died") {
			canNotAttack[addr] = true
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
