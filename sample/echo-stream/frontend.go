package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	echo "github.com/appnet-org/golib/sample/echo-stream/pb"
	"google.golang.org/grpc"
)

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "server:9000", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		http.Error(w, "Failed to connect to gRPC server", http.StatusInternalServerError)
		log.Printf("Failed to connect to gRPC server: %v", err)
		return
	}
	defer conn.Close()

	grpcClient := echo.NewEchoServiceClient(conn)
	stream, err := grpcClient.Echo(context.Background())
	if err != nil {
		http.Error(w, "Failed to establish stream", http.StatusInternalServerError)
		log.Printf("Failed to establish stream: %v", err)
		return
	}
	defer stream.CloseSend()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				in, err := stream.Recv()
				if err == io.EOF {
					return
				}
				if err != nil {
					log.Printf("Failed to receive a message: %v", err)
					return
				}
				log.Printf("Got message: %s", in.Body)
			}
		}
	}()

	requestBody := r.URL.Query().Get("key")
	fmt.Printf("Frontend got request with key: %s\n", requestBody)

	if requestBody == "" {
		http.Error(w, "key parameter is required", http.StatusBadRequest)
		return
	}

	for i := 0; i < 1; i++ {
		msg := fmt.Sprintf("%s %d", requestBody, i+1)
		if err := stream.Send(&echo.Msg{Body: msg}); err != nil {
			log.Printf("Failed to send a message: %v", err)
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			return
		}
		log.Printf("Sent message: %s", msg)
		time.Sleep(1 * time.Second) // Simulate some delay
	}

	fmt.Fprintf(w, "Sent message: %s\n", requestBody)
}

func main() {
	http.HandleFunc("/", handleHTTP)
	server := &http.Server{
		Addr: ":8080",
	}

	go func() {
		log.Println("HTTP server listening on port 8080!!!")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server Shutdown: %v", err)
	}
}
