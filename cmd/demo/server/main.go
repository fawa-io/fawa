package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	hellov1 "github.com/fawa-io/fawa/gen/hello/v1"
	"github.com/fawa-io/fawa/gen/hello/v1/hellov1connect"
)

type HelloServer struct{}

func (s *HelloServer) SayHello(
	ctx context.Context,
	req *connect.Request[hellov1.SayHelloRequest],
) (*connect.Response[hellov1.SayHelloResponse], error) {
	log.Println("Request headers: ", req.Header())
	res := connect.NewResponse(&hellov1.SayHelloResponse{
		Resp: fmt.Sprintf("Hello, %s!", req.Msg.Name),
	})
	res.Header().Set("Hello-Version", "v1")
	return res, nil
}

func main() {
	server := &HelloServer{}
	mux := http.NewServeMux()
	path, handler := hellov1connect.NewHelloServiceHandler(server)
	mux.Handle(path, handler)
	log.Println("Starting hello demo server on :8081")
	http.ListenAndServe(
		"localhost:8081", // Use a different port for the demo server
		h2c.NewHandler(mux, &http2.Server{}),
	)
}
