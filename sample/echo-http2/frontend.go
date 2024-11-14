package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	echo "github.com/appnet-org/golib/sample/echo-pb" // Update this import path as needed
	"github.com/golang/protobuf/proto"                // Import for Protobuf serialization/deserialization
	"golang.org/x/net/http2"
)

func handler(writer http.ResponseWriter, request *http.Request) {
	requestBody := request.URL.Query().Get("key")
	// fmt.Printf("Frontend received request with key: %s\n", requestBody)

	message := &echo.Msg{
		Body: requestBody,
	}

	data, err := proto.Marshal(message)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error serializing message: %v", err), http.StatusInternalServerError)
		return
	}

	// Use a custom HTTP/2 client that supports h2c (unencrypted HTTP/2)
	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true, // Enable HTTP/2 over HTTP (no TLS)
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}

	req, err := http.NewRequest("POST", "http://localhost:9000", bytes.NewReader(data)) // Use http (no TLS)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error creating HTTP request: %v", err), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	// req.Header.Set("Upgrade", "h2c") // Explicitly indicate the intent to use h2c (optional but may help with some servers)

	resp, err := client.Do(req)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error sending request to server: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error reading response: %v", err), http.StatusInternalServerError)
		return
	}

	var response echo.Msg
	err = proto.Unmarshal(respData, &response)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Error deserializing response: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(writer, "Response from HTTP/2 server: %s\n", response.GetBody())
	// log.Printf("Response from server: %s", response.GetBody())
}

func main() {
	http.HandleFunc("/", handler)

	fmt.Printf("Frontend listening on port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
