package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

var players = make(map[*net.UDPAddr]string)

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
			players[udpAddr] = username
			fmt.Println(string(buffer[:n]))
			break
		}
	}

	go receiveMessages(udpAddr, conn)

	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		_, err := conn.Write([]byte(text))
		if err != nil {
			fmt.Println("Error sending message:", err)
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
			fmt.Println(string(buffer[:n]))
			os.Exit(0)
			break
		}

		if strings.HasPrefix(response, "POKEMON_LIST|") {
			reader := bufio.NewReader(os.Stdin)

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

			chooseMsg := fmt.Sprintf("CHOOSE|%s|%s|%s", pokemon1, pokemon2, pokemon3)
			_, err = conn.Write([]byte(chooseMsg))
			if err != nil {
				panic(err)
			}

			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				panic(err)
			}

			response := string(buffer[:n])
			if response == "CHOSEN|"+players[addr] {
				fmt.Println("Pokémon chosen successfully! Waiting for battle...")

				for {
					fmt.Print("Enter command (/ATTACK/CHANGE [pokemon]): ")
					cmd, _ := reader.ReadString('\n')
					cmd = strings.TrimSpace(cmd)

					_, err := conn.Write([]byte(cmd + "|" + players[addr]))
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
			break
		}

		fmt.Println(string(buffer[:n]))
	}
}
