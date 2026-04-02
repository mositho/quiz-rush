FROM golang:1.25.8-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/questions-server ./cmd/server

FROM alpine:3.22
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/questions-server /app/questions-server
COPY questionsets /app/questionsets

ENV QUESTION_SETS_DIR=/app/questionsets

ENTRYPOINT ["/app/questions-server"]
