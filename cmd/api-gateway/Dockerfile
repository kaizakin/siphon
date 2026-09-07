# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api-gateway ./cmd/api-gateway

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/api-gateway /usr/local/bin/api-gateway
EXPOSE 5000
ENTRYPOINT ["/usr/local/bin/api-gateway"]
