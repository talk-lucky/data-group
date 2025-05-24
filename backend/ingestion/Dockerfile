# Use an official Go runtime as a parent image
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files (assuming Docker context is backend/ingestion/)
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download && go mod verify

# Copy the source code into the container
# Using *.go . as typical for Go projects where all .go files in the dir are part of main package
COPY *.go .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/ingestion-service .

# Start a new stage from scratch
FROM alpine:latest

# Add CA certificates
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/ingestion-service .

# Expose port (e.g., 8081) to the outside world
EXPOSE 8081

# Command to run the executable
CMD ["./ingestion-service"]
