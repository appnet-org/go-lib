package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	echo "github.com/appnet-org/golib/sample/echo-pb"
	"github.com/golang/protobuf/proto"
)

func handler(writer http.ResponseWriter, request *http.Request) {
	requestBody := request.URL.Query().Get("key")
	// fmt.Printf("Frontend received request with key: %s\n", requestBody)

	serverAddr := net.UDPAddr{
		Port: 9000,
		IP:   net.ParseIP("127.0.0.1"), // Change IP if needed
	}

	conn, err := net.DialUDP("udp", nil, &serverAddr)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Could not connect to UDP server: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	message := &echo.Msg{
		Body: requestBody,
	}

	data, err := proto.Marshal(message)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error serializing message: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error sending data: %v", err), http.StatusInternalServerError)
		return
	}

	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // Set a timeout for the response
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error receiving response: %v", err), http.StatusInternalServerError)
		return
	}

	var response echo.Msg
	err = proto.Unmarshal(buffer[:n], &response)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error deserializing response: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(writer, "Response from UDP server: %s\n", response.GetBody())
	// log.Printf("Response from server: %s", response.GetBody())
}

func main() {
	http.HandleFunc("/", handler)

	fmt.Printf("Frontend listening on port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
