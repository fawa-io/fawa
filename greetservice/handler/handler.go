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

package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"connectrpc.com/connect"
	"github.com/fawa-io/fwpkg/fwlog"

	greetv1 "github.com/fawa-io/fawa/greetservice/gen/greet/v1"
)

type GreetServiceHandler struct{}

func (s *GreetServiceHandler) SayHello(
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
func (s *GreetServiceHandler) GreetStream(
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
	}
	return nil
}

func (s *GreetServiceHandler) GreetClientStream(
	ctx context.Context,
	stream *connect.ClientStream[greetv1.GreetClientStreamRequest],
) (*connect.Response[greetv1.GreetClientStreamResponse], error) {
	var names []string
	for stream.Receive() {
		fwlog.Debugf("cilent stream receive: %v", stream.Msg().Name)
		names = append(names, stream.Msg().Name)
	}

	if err := stream.Err(); err != nil {
		fwlog.Errorf("Stream ended with an error: %v", err)
		return nil, connect.NewError(connect.CodeUnknown, err)
	}

	fwlog.Debugf("Stream finished successfully. Received names: %v", names)

	resp := connect.NewResponse(&greetv1.GreetClientStreamResponse{
		Summary: fmt.Sprintf("Hello, %s!", strings.Join(names, ", ")),
	})
	return resp, nil
}

// GreetBidiStream implements the bidirectional-streaming RPC.
func (s *GreetServiceHandler) GreetBidiStream(
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
				fwlog.Debug("bidi stream finished successfully.")
				return nil
			}
			return err
		}
		fwlog.Debugf("bidi stream receive: %v", req.Name)
		if err := stream.Send(&greetv1.GreetBidiStreamResponse{
			Echo: fmt.Sprintf("Hello, %s!", req.Name),
		}); err != nil {
			return err
		}
	}
}
