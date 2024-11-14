package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	echo "github.com/appnet-org/golib/sample/echo-pb" // Update this import path as needed
	"github.com/golang/protobuf/proto"                // Import for Protobuf serialization/deserialization
)

func handler(writer http.ResponseWriter, request *http.Request) {
	requestBody := request.URL.Query().Get("key")
	// fmt.Printf("Frontend received request with key: %s\n", requestBody)

	conn, err := net.Dial("tcp", "127.0.0.1:9000") // Establish a TCP connection to the server
	if err != nil {
		http.Error(writer, fmt.Sprintf("Could not connect to TCP server: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	message := &echo.Msg{
		Body: requestBody + "1",
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

	// Read response
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // Set a timeout for the response
	n, err := conn.Read(buffer)
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

	fmt.Fprintf(writer, "Response from TCP server: %s\n", response.GetBody())
	// log.Printf("Response from server: %s", response.GetBody())
}

func main() {
	http.HandleFunc("/", handler)

	fmt.Printf("Frontend listening on port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
