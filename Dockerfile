# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod ./
RUN go mod download

COPY . .
RUN go test ./... && \
    go build -trimpath -ldflags="-s -w" -o /out/subscription-billing-api ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai \
    PORT=8080 \
    DATA_FILE=/app/data/subscriptions.json

WORKDIR /app
COPY --from=builder /out/subscription-billing-api /app/subscription-billing-api
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/subscription-billing-api"]
