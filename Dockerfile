FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/app ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/seeder ./cmd/seeder

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/app .
COPY --from=builder /app/seeder .
COPY --from=builder /build/migrations ./migrations

CMD ["./app"]

