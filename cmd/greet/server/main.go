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
	"fmt"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"connectrpc.com/connect"

	"github.com/fawa-io/fawa/gen/greet/v1"
	"github.com/fawa-io/fawa/gen/greet/v1/greetv1connect"
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

func main() {
	srv := &GreetServer{}
	mux := http.NewServeMux()
	procedure, hdr := greetv1connect.NewGreetServiceHandler(srv)
	mux.Handle(procedure, hdr)
	fwlog.Info("Starting greet server on :8081")
	fwlog.Fatal(http.ListenAndServe(
		"localhost:8081", // Use a different port for the demo server
		h2c.NewHandler(mux, &http2.Server{}),
	))
}
