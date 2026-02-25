FROM alpine:3.19

WORKDIR /app

# Copy pre-built binary from goreleaser
COPY mk-addrlist-generator .

# Create directory for configuration
RUN mkdir -p /etc/mk-addrlist-generator
EXPOSE 8080
ENV GIN_MODE=release

# Set the binary as the entrypoint
ENTRYPOINT ["/app/mk-addrlist-generator"]
CMD ["--config", "/etc/mk-addrlist-generator/config.yaml"]