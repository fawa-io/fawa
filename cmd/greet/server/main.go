package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"connectrpc.com/connect"

	"github.com/fawa-io/fawa/gen/greet/v1"
	"github.com/fawa-io/fawa/gen/greet/v1/greetv1connect"
)

type GreetServer struct{}

func (s *GreetServer) SayHello(
	ctx context.Context,
	req *connect.Request[greetv1.SayHelloRequest],
) (*connect.Response[greetv1.SayHelloResponse], error) {
	log.Println("Request headers: ", req.Header())
	res := connect.NewResponse(&greetv1.SayHelloResponse{
		Resp: fmt.Sprintf("Hello, %s!", req.Msg.Name),
	})
	res.Header().Set("Greet-Version", "v1")
	return res, nil
}

func main() {
	srv := &GreetServer{}
	mux := http.NewServeMux()
	procedure, hdr := greetv1connect.NewGreetServiceHandler(srv)
	mux.Handle(procedure, hdr)
	log.Println("Starting greet server on :8081")
	log.Fatal(http.ListenAndServe(
		"localhost:8081", // Use a different port for the demo server
		h2c.NewHandler(mux, &http2.Server{}),
	))
}
