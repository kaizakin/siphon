# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.27.3

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations
ENTRYPOINT ["sh", "-c", "goose -dir /migrations postgres \"$DATABASE_URL\" up"]
