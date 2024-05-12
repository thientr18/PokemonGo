package main

import (
	"fmt"
	"net"
)

const (
	HOST = "localhost"
	PORT = "8080"
	TYPE = "udp"
)

func main() {
	udpAddr, err := net.ResolveUDPAddr(TYPE, HOST+":"+PORT)
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
}
