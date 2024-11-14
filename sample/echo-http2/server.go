package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	echo "github.com/appnet-org/golib/sample/echo-pb" // Update this import path as needed
	"github.com/golang/protobuf/proto"                // Import for Protobuf serialization/deserialization
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type server struct {
	requestCount uint64 // Use an atomic uint64 to track the request count
}

func (s *server) handleRequest(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error reading request body: %v", err), http.StatusInternalServerError)
		return
	}
	defer request.Body.Close()

	var msg echo.Msg
	err = proto.Unmarshal(body, &msg)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error deserializing message: %v", err), http.StatusInternalServerError)
		return
	}

	// Increment request count
	atomic.AddUint64(&s.requestCount, 1)

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
		http.Error(writer, fmt.Sprintf("Error serializing response: %v", err), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Write(responseData)
}

func main() {
	srv := &server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleRequest)

	server := &http.Server{
		Addr:    ":9000",
		Handler: h2c.NewHandler(mux, &http2.Server{}), // Configure for h2c (HTTP/2 without TLS)
	}

	log.Printf("HTTP/2 (h2c) server listening on port 9000")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
