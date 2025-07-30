# Copyright 2025 The fawa Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# build
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /server -ldflags "-s -w" .

# run
FROM alpine:latest

RUN apk add --no-cache ca-certificates

RUN addgroup -S agroup && adduser -S auser -G agroup
USER auser

WORKDIR /app

COPY --from=builder --chown=auser:agroup /server .


EXPOSE 443

CMD ["./server"]
