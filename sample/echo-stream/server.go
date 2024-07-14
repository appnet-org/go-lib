package main

import (
	"io"
	"log"
	"net"
	"time"

	echo "github.com/appnet-org/golib/sample/echo-stream/pb"
	"google.golang.org/grpc"
)

type server struct {
	echo.UnimplementedEchoServiceServer
}

func (s *server) Echo(stream echo.EchoService_EchoServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("Received message: %s", msg.Body)

		// Send two responses for each request
		for i := 0; i < 2; i++ {
			responseMsg := &echo.Msg{Body: "Echo " + string(i+1) + ": " + msg.Body}
			err = stream.Send(responseMsg)
			if err != nil {
				return err
			}
			time.Sleep(100 * time.Millisecond) // Simulate some delay
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	echo.RegisterEchoServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
