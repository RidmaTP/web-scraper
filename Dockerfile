FROM golang:1.26.0-alpine AS builder

RUN apk add --no-cache chromium ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download


COPY . .

RUN go build -o /web-scraper ./cmd/server


EXPOSE 8000

CMD ["/web-scraper"]