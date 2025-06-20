package main

import (
	"context"
	"log"
	"net/http"

	"connectrpc.com/connect"

	greetv1 "github.com/fawa-io/fawa/gen/greet/v1"
	"github.com/fawa-io/fawa/gen/greet/v1/greetv1connect"
)

func main() {
	cli := greetv1connect.NewGreetServiceClient(
		http.DefaultClient,
		"http://localhost:8081",
	)
	res, err := cli.SayHello(
		context.Background(),
		connect.NewRequest(&greetv1.SayHelloRequest{Name: "World"}),
	)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(res.Msg.Resp)
}
