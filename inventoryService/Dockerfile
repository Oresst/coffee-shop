FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o inventory-service cmd/main.go


FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/inventory-service .
COPY --from=builder /app/.docker_env .env

EXPOSE 8080

CMD ["./inventory-service"]
