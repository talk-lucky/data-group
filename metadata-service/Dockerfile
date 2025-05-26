# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
# -ldflags="-w -s" reduces the size of the binary by removing debug information
# CGO_ENABLED=0 disables Cgo, which is needed for a static build when using Alpine
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o metadata-service cmd/server/main.go

# Stage 2: Create the final, minimal image
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/metadata-service .
COPY --from=builder /app/.env.example .env.example
COPY --from=builder /app/db/migrations/init.sql ./db/migrations/init.sql
# Note: In a real application, you might not want to copy .env.example directly
# or init.sql if your migrations are handled by a separate tool or within the app at startup.
# For this example, we include them.

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
# The server will read its configuration from environment variables,
# which can be supplied when running the Docker container.
# e.g. docker run -p 8080:8080 -e DB_HOST=my_db_host ... metadata-service
CMD ["./metadata-service"]
