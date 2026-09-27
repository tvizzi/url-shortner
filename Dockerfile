# b
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /url-shortener ./cmd/url-shortener

# r
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /url-shortener ./url-shortener
COPY config ./config

EXPOSE 8082

ENTRYPOINT ["./url-shortener"]
