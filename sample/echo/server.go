package main

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"

	interceptor "github.com/appnet-org/golib/interceptor"
	echo "github.com/appnet-org/golib/sample/echo-pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type server struct {
	echo.UnimplementedEchoServiceServer
	requestCount uint64 // Use an atomic uint64 to track the request count
}

func (s *server) Echo(ctx context.Context, x *echo.Msg) (*echo.Msg, error) {
	// Atomically increment the request count
	atomic.AddUint64(&s.requestCount, 1)

	// Log the HTTP headers received
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		log.Println("Received HTTP Headers:")
		for key, values := range md {
			log.Printf("  %s: %v", key, values)
		}
	} else {
		log.Println("No metadata (HTTP headers) received.")
	}

	log.Printf("Server got: [%s]", x.GetBody())

	// Check if the message contains "sleep"
	if x.GetBody() == "sleep" {
		log.Printf("Sleeping for 30 seconds...")
		time.Sleep(30 * time.Second)
	}

	msg := &echo.Msg{
		Body: x.GetBody(),
	}

	return msg, nil
}

func (s *server) logRequestCount() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Atomically load the request count
		count := atomic.LoadUint64(&s.requestCount)
		log.Printf("Total Echo requests received: %d", count)
	}
}

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := &server{}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ServerInterceptor("/appnet/interceptors/server")))

	// Start the request count logging in a separate goroutine
	go srv.logRequestCount()

	fmt.Printf("Starting server pod at port 9000\n")

	echo.RegisterEchoServiceServer(grpcServer, srv)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
