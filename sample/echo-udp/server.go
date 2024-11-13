package main

import (
	"log"
	"net"
	"time"

	echo "github.com/appnet-org/golib/sample/echo-pb"
	"github.com/golang/protobuf/proto"
)

type server struct {
}

func (s *server) handleRequest(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	var msg echo.Msg
	err := proto.Unmarshal(data, &msg)
	if err != nil {
		log.Printf("Error deserializing message: %v", err)
		return
	}

	// log.Printf("Server received: [%s]", msg.GetBody())

	// Check if the message contains "sleep"
	if msg.GetBody() == "sleep" {
		log.Printf("Sleeping for 30 seconds...")
		time.Sleep(30 * time.Second)
	}

	response := &echo.Msg{
		Body: msg.GetBody(),
	}

	responseData, err := proto.Marshal(response)
	if err != nil {
		log.Printf("Error serializing response: %v", err)
		return
	}

	_, err = conn.WriteToUDP(responseData, addr)
	if err != nil {
		log.Printf("Error sending response: %v", err)
	}
}

func main() {
	addr := net.UDPAddr{
		Port: 9000,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("failed to listen on UDP port 9000: %v", err)
	}
	defer conn.Close()

	srv := &server{}

	log.Printf("UDP server listening on port 9000")
	buffer := make([]byte, 4096)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error receiving data: %v", err)
			continue
		}

		go srv.handleRequest(conn, clientAddr, buffer[:n])
	}
}
