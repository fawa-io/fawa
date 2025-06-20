package main

import (
	"context"
	"log"
	"net/http"

	"connectrpc.com/connect"

	hellov1 "github.com/fawa-io/fawa/gen/hello/v1"
	"github.com/fawa-io/fawa/gen/hello/v1/hellov1connect"
)

func main() {
	client := hellov1connect.NewHelloServiceClient(
		http.DefaultClient,
		"http://localhost:8081", // Point to the demo server
	)
	res, err := client.SayHello(
		context.Background(),
		connect.NewRequest(&hellov1.SayHelloRequest{Name: "Demo"}),
	)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(res.Msg.Resp)
}
