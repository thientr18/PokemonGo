package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:8080")
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your player ID: ")
	playerID, _ := reader.ReadString('\n')
	playerID = strings.TrimSpace(playerID)

	joinMsg := fmt.Sprintf("JOIN|%s", playerID)
	_, err = conn.Write([]byte(joinMsg))
	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		panic(err)
	}

	response := string(buffer[:n])
	if strings.HasPrefix(response, "POKEMON_LIST|") {
		fmt.Println("Available Pokémon:")
		fmt.Println(strings.TrimPrefix(response, "POKEMON_LIST|"))

		fmt.Print("Choose your first Pokémon: ")
		pokemon1, _ := reader.ReadString('\n')
		pokemon1 = strings.TrimSpace(pokemon1)

		fmt.Print("Choose your second Pokémon: ")
		pokemon2, _ := reader.ReadString('\n')
		pokemon2 = strings.TrimSpace(pokemon2)

		fmt.Print("Choose your third Pokémon: ")
		pokemon3, _ := reader.ReadString('\n')
		pokemon3 = strings.TrimSpace(pokemon3)

		chooseMsg := fmt.Sprintf("CHOOSE|%s|%s|%s|%s", playerID, pokemon1, pokemon2, pokemon3)
		_, err = conn.Write([]byte(chooseMsg))
		if err != nil {
			panic(err)
		}

		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			panic(err)
		}

		response := string(buffer[:n])
		if response == "CHOSEN|"+playerID {
			fmt.Println("Pokémon chosen successfully! Waiting for battle...")

			for {
				fmt.Print("Enter command (INVITE [playerID]/ACCEPT [playerID]/ATTACK/CHANGE [pokemon]): ")
				cmd, _ := reader.ReadString('\n')
				cmd = strings.TrimSpace(cmd)

				_, err := conn.Write([]byte(cmd + "|" + playerID))
				if err != nil {
					panic(err)
				}

				n, _, err := conn.ReadFromUDP(buffer)
				if err != nil {
					panic(err)
				}

				response := string(buffer[:n])
				fmt.Println("Response from server:", response)

				if strings.HasPrefix(response, "WIN|") {
					fmt.Println("You win!")
					break
				} else if strings.HasPrefix(response, "START|") {
					fmt.Println("Battle started!")
				}
			}
		} else {
			fmt.Println("Failed to choose Pokémon:", response)
		}
	} else {
		fmt.Println("Failed to join the game:", response)
	}
}
