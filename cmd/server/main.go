package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	filev1 "github.com/fawa-io/fawa/gen/file/v1"
	"github.com/fawa-io/fawa/gen/file/v1/filev1connect"
)

type SayHelloServer struct{}

// use generics
func (s *SayHelloServer) SayHello(ctx context.Context,
	req *connect.Request[filev1.SayHelloRequest]) (*connect.Response[filev1.SayHelloResponse], error) {
	log.Println("Request headers: ", req.Header())
	res := connect.NewResponse(&filev1.SayHelloResponse{
		Resp: fmt.Sprintf("Hello, %s!", req.Msg.Name),
	})
	res.Header().Set("File-Version", "v1")
	return res, nil
}

func main() {
	sayhello := &SayHelloServer{}
	mux := http.NewServeMux()
	path, handler := filev1connect.NewHelloServiceHandler(sayhello)
	mux.Handle(path, handler)
	http.ListenAndServe("localhost:8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)
}
