// Copyright 2025 The fawa Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"net/http"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"connectrpc.com/connect"

	greetv1 "github.com/fawa-io/fawa/gen/proto/fawa/greet/v1"
	"github.com/fawa-io/fawa/gen/proto/fawa/greet/v1/greetv1connect"
	"github.com/fawa-io/fawa/pkg/fwlog"
)

type GreetServer struct{}

func (s *GreetServer) SayHello(
	ctx context.Context,
	req *connect.Request[greetv1.SayHelloRequest],
) (*connect.Response[greetv1.SayHelloResponse], error) {
	fwlog.Debugf("Request headers: %v", req.Header())
	res := connect.NewResponse(&greetv1.SayHelloResponse{
		Resp: fmt.Sprintf("Hello, %s!", req.Msg.Name),
	})
	res.Header().Set("Greet-Version", "v1")
	return res, nil
}

// GreetStream implements the server-streaming RPC.
func (s *GreetServer) GreetStream(
	ctx context.Context,
	req *connect.Request[greetv1.GreetStreamRequest],
	stream *connect.ServerStream[greetv1.GreetStreamResponse],
) error {
	name := req.Msg.Name
	if name == "" {
		name = "World"
	}
	for i := 0; i < 10; i++ {
		if err := stream.Send(&greetv1.GreetStreamResponse{
			Part: fmt.Sprintf("Hello, %s! (part %d)", name, i+1),
		}); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

// GreetClientStream implements the client-streaming RPC.
func (s *GreetServer) GreetClientStream(
	ctx context.Context,
	stream *connect.ClientStream[greetv1.GreetClientStreamRequest],
) (*connect.Response[greetv1.GreetClientStreamResponse], error) {
	var names []string
	for stream.Receive() {
		names = append(names, stream.Msg().Name)
	}
	if err := stream.Err(); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, err)
	}
	resp := connect.NewResponse(&greetv1.GreetClientStreamResponse{
		Summary: fmt.Sprintf("Hello, %s!", strings.Join(names, ", ")),
	})
	return resp, nil
}

// GreetBidiStream implements the bidirectional-streaming RPC.
func (s *GreetServer) GreetBidiStream(
	ctx context.Context,
	stream *connect.BidiStream[greetv1.GreetBidiStreamRequest, greetv1.GreetBidiStreamResponse],
) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		req, err := stream.Receive()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := stream.Send(&greetv1.GreetBidiStreamResponse{
			Echo: fmt.Sprintf("Hello, %s!", req.Name),
		}); err != nil {
			return err
		}
	}
}

func main() {
	srv := &GreetServer{}
	mux := http.NewServeMux()
	path, handler := greetv1connect.NewGreetServiceHandler(srv)
	mux.Handle(path, handler)
	fwlog.Info("Starting greet server on :8081")
	// Use h2c so we can serve HTTP/2 without TLS.

	if err := http.ListenAndServe("localhost:8081", h2c.NewHandler(newCORS().Handler(mux), &http2.Server{})); err != nil {
		fwlog.Error(err)
	}
}
