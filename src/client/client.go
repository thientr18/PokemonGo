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

		if strings.HasPrefix(response, "@pokemonlist") {
			fmt.Print("Available pokemon:")
			fmt.Println(strings.TrimPrefix(response, "@pokemonlist"))
			break
		}

		if response == "Battle Started!" {
			reader := bufio.NewReader(os.Stdin)

			fmt.Println("Want to see your pokemons? \nYes[y]\nNo[n]")
			list, _ := reader.ReadString('\n')
			list = strings.TrimSpace(list)
			if list == "y" {
				n, _, err := conn.ReadFromUDP(buffer)
				if err != nil {
					panic(err)
				}
				response := string(buffer[:n])

				fmt.Print("Available pokemon:")
				fmt.Println(strings.TrimPrefix(response, "@pokemonlist"))
			} else if list == "n" {
				continue
			} else {
				fmt.Println("Invalid command")
			}

			fmt.Print("Choose your first pokemon: ")
			pokemon1, _ := reader.ReadString('\n')
			pokemon1 = strings.TrimSpace(pokemon1)

			fmt.Print("Choose your second pokemon: ")
			pokemon2, _ := reader.ReadString('\n')
			pokemon2 = strings.TrimSpace(pokemon2)

			fmt.Print("Choose your third pokemon: ")
			pokemon3, _ := reader.ReadString('\n')
			pokemon3 = strings.TrimSpace(pokemon3)

			chooseMsg := fmt.Sprintf("@choose %s %s %s", pokemon1, pokemon2, pokemon3)
			_, err := conn.Write([]byte(chooseMsg))
			if err != nil {
				panic(err)
			}

			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				panic(err)
			}

			response := string(buffer[:n])
			if response == "@chosen"+players[addr] {
				fmt.Println("pokemon chosen successfully! Waiting for battle...")

				for {
					fmt.Print("Enter command (@acttack/@change [pokemon]): ")
					cmd, _ := reader.ReadString('\n')
					cmd = strings.TrimSpace(cmd)

					_, err := conn.Write([]byte(cmd + " " + players[addr]))
					if err != nil {
						panic(err)
					}

					n, _, err := conn.ReadFromUDP(buffer)
					if err != nil {
						panic(err)
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
				}
			} else {
				fmt.Println("Failed to choose pokemon:", response)
			}
			break
		}

		fmt.Println(string(buffer[:n]))
	}
}
