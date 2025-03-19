FROM golang:1.23.4-alpine AS builder

WORKDIR /app

# Download deps
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN mkdir /app/bin
RUN go build -o /app/bin/build ./cmd/server/main.go

FROM alpine:latest

WORKDIR /app
RUN mkdir /app/bin
COPY --from=builder /app/bin/build /app/bin/build

CMD ["/app/bin/build"]