# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth-svc ./cmd/auth-svc

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/auth-svc /usr/local/bin/auth-svc
EXPOSE 5002
ENTRYPOINT ["/usr/local/bin/auth-svc"]
