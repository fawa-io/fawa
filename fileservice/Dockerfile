# build
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLE=0 go build -o /server -ldflags "-s -w"

# run
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /server .

COPY ./config.yaml .
COPY ./cert ./cert/

EXPOSE 8081

CMD ["./server"]