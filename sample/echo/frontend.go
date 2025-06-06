package main

import (
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	interceptor "github.com/appnet-org/golib/interceptor"
	echo "github.com/appnet-org/golib/sample/echo-pb"
)

func handler(writer http.ResponseWriter, request *http.Request) {
	// requestBody := strings.Replace(request.URL.String(), "/", "", -1)
	requestBody := request.URL.Query().Get("key")
	fmt.Printf("Frontend got request with key: %s !\n", requestBody)

	var conn *grpc.ClientConn

	conn, err := grpc.Dial(
		"server:9000",
		grpc.WithUnaryInterceptor(interceptor.ClientInterceptor("/appnet/interceptors/frontend", "/appnet/interceptors/lb")),
		// grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"appnet_lb":{}}]}`),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("could not connect: %s", err)
	}
	defer conn.Close()

	c := echo.NewEchoServiceClient(conn)

	// Create and attach metadata with the custom header
	md := metadata.New(map[string]string{
		"key": requestBody, // Here we're setting the custom header "key" to the requestBody
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	message := echo.Msg{
		Body: requestBody,
	}

	var header metadata.MD

	// Make sure to pass the context (ctx) which includes the metadata
	response, err := c.Echo(ctx, &message, grpc.Header(&header))

	if err != nil {
		fmt.Fprintf(writer, "Echo server returns an error: %s\n", err)
		log.Printf("Error when calling echo: %s", err)
	} else {
		fmt.Fprintf(writer, "Response from server: %s\n", response.Body)
		log.Printf("Response from server: %s", response.Body)

		// Print the response headers (metadata)
		log.Println("Response headers:")
		for key, values := range header {
			log.Printf("  %s: %v", key, values)
		}
	}
}

func main() {
	http.HandleFunc("/", handler)

	fmt.Printf("Starting frontend pod at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
