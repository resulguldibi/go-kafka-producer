# Build stage
FROM golang:1.22-alpine3.20 AS builder

WORKDIR /app

# Update and install git for go modules
RUN apk update && apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:3.20

# Update and install ca-certificates for HTTPS requests
RUN apk update && apk upgrade && apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Set environment variables with default values
ENV PORT=8080
ENV KAFKA_BROKERS=localhost:9092

# Run the application
CMD ["./main"]
