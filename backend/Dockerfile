# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./main.go

# Final Stage
FROM alpine:latest

WORKDIR /app

# Install CA certificates for HTTPS requests (e.g., to Gemini API)
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Create a non-root user (Hugging Face recommendation)
RUN adduser -D -u 1000 user
USER user

# Set environment variables for production
ENV GIN_MODE=release
# Hugging Face Spaces require port 7860
ENV PORT=7860

EXPOSE 7860

# Run the binary
CMD ["./main"]
