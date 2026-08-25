# --- Stage 1: Builder ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o order-service cmd/main.go


# --- Stage 2: Runner (Минимальный образ) ---
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/order-service .
COPY --from=builder /app/.env .

EXPOSE 8080

CMD ["./order-service"]