package main

import (
	"connectrpc.com/connect"

	filev1 "github.com/fawa-io/fawa/gen/file/v1"
	"github.com/fawa-io/fawa/gen/file/v1/filev1connect"

	"context"
	"log"
	"net/http"
)

func main() {
	client := filev1connect.NewHelloServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)
	res, err := client.SayHello(
		context.Background(),
		connect.NewRequest(&filev1.SayHelloRequest{Name: "FAWA"}),
	)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(res.Msg.Resp)
}
