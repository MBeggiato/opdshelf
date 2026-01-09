FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o opds-server ./cmd/server

# Use a smaller image for the final container
FROM alpine:latest

# Install certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/opds-server /app/opds-server

# Copy templates and static files
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/static /app/static

# Create books directory
RUN mkdir -p /app/books

# Set environment variables
ENV PORT=3000
ENV HOST=0.0.0.0
ENV BOOKS_DIR=/app/books

# Expose the port
EXPOSE 3000

# Create volume for persistent storage of books
VOLUME ["/app/books"]

# Run the application
CMD ["/app/opds-server"]
