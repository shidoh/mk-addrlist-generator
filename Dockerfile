FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o mk-addrlist-generator

FROM alpine:3.19

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/mk-addrlist-generator .

# Create directory for configuration
RUN mkdir -p /etc/mk-addrlist-generator
EXPOSE 8080
ENV GIN_MODE=release
# Set the binary as the entrypoint
ENTRYPOINT ["/app/mk-addrlist-generator"]
CMD ["--config", "/etc/mk-addrlist-generator/config.yaml"]