# Build stage
FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API and Seeder binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/seeder ./cmd/seeder

# Final runtime stage
FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/seeder /app/seeder
COPY --from=builder /app/db/migrations /app/db/migrations
COPY --from=builder /app/.env.example /app/.env.example

# Create uploads directory
RUN mkdir -p /app/uploads/payments

EXPOSE 8080

CMD ["/app/api"]
