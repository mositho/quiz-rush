FROM golang:1.25.8-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/game-server ./cmd/server

FROM alpine:3.22
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/game-server /app/game-server
COPY migrations /app/migrations

ENTRYPOINT ["/app/game-server"]
