package main

import (
	"fmt"
	"net"
	"strings"
)

type Client struct {
	Name string
	Addr *net.UDPAddr
}

var clients = make(map[string]*Client)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "localhost:8080")
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Server is running on", udpAddr)

	buffer := make([]byte, 1024)

	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading:", err)
			continue
		}

		message := string(buffer[:n])
		handleMessage(message, addr, conn)
	}
}

func handleMessage(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	if strings.HasPrefix(message, "@") {
		parts := strings.SplitN(message, " ", 3)
		command := parts[0]
		senderName := getUsernameByAddr(addr) // Get sender's name

		switch command {
		case "@join":
			if !checkExistedClient(parts[1]) {
				sendMessageToClient("duplicated-username", addr, conn)
			} else {
				username := parts[1]
				clients[username] = &Client{Name: username, Addr: addr}
				fmt.Printf("User '%s' joined\n", username)
				sendMessageToClient("Welcome to the chat, "+username+"!", addr, conn)
			}
		case "@all":
			allParts := strings.SplitN(message, " ", 2)     // allParts is parts of message for @all comamnd
			broadcastMessage(allParts[1], senderName, conn) // Pass sender's name
		case "@quit":
			username := getUsernameByAddr(addr)
			delete(clients, username)
			fmt.Printf("User '%s' left\n", username)
			sendMessageToClient("Goodbye, "+username+"!", addr, conn)
		case "@private":
			recipient := parts[1]
			if _, ok := clients[recipient]; !ok {
				sendErrorMessageToClient("Error: Recipient did not exist in the server!", addr, conn)
				// to send a error message to the sender
			}
			privateMessage := senderName + " (private): " + parts[2]
			sendPrivateMessage(privateMessage, recipient, conn)
		default:
			sendMessageToClient("Invalid command", addr, conn)
		}
	} else {
		sendMessageToClient("Invalid command format", addr, conn)
	}
}

func checkExistedClient(username string) bool {
	_, exists := clients[username]
	if !exists {
		return true
	} else {
		return false
	}
}

func getUsernameByAddr(addr *net.UDPAddr) string {
	for _, client := range clients {
		if client.Addr.IP.Equal(addr.IP) && client.Addr.Port == addr.Port {
			return client.Name
		}
	}
	return ""
}

func sendMessageToClient(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	_, err := conn.WriteToUDP([]byte(message), addr)
	if err != nil {
		fmt.Println("Error sending message:", err)
	}
}

func broadcastMessage(message string, senderName string, conn *net.UDPConn) {
	for username, client := range clients {
		if username != senderName {
			fullMessage := senderName + " (public): " + message // Include sender's name
			_, err := conn.WriteToUDP([]byte(fullMessage), client.Addr)
			if err != nil {
				fmt.Println("Error broadcasting message:", err)
			}
		}
	}
}

func sendPrivateMessage(message, recipient string, conn *net.UDPConn) {
	client, exists := clients[recipient]
	if !exists {
		fmt.Println("Recipient not found:", recipient)
		return
	}
	_, err := conn.WriteToUDP([]byte(message), client.Addr)
	if err != nil {
		fmt.Println("Error sending private message:", err)
	}
}

func sendErrorMessageToClient(message string, addr *net.UDPAddr, conn *net.UDPConn) {
	_, err := conn.WriteToUDP([]byte(message), addr)
	if err != nil {
		fmt.Println("Error sending error message:", err)
	}
}
