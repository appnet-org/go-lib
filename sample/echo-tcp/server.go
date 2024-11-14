package main

import (
	"log"
	"net"
	"time"

	echo "github.com/appnet-org/golib/sample/echo-pb" // Update this import path as needed
	"github.com/golang/protobuf/proto"                // Import for Protobuf serialization/deserialization
)

type server struct {
}

func (s *server) handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading data: %v", err)
		return
	}

	var msg echo.Msg
	err = proto.Unmarshal(buffer[:n], &msg)
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

	_, err = conn.Write(responseData)
	if err != nil {
		log.Printf("Error sending response: %v", err)
	}
}

func main() {
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen on TCP port 9000: %v", err)
	}
	defer listener.Close()

	srv := &server{}

	log.Printf("TCP server listening on port 9000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go srv.handleConnection(conn)
	}
}
