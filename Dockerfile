# Use the official Go image for 1.25.4
FROM golang:1.25.4

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first to leverage caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
RUN go build -tags netgo -ldflags "-s -w" -o app

# Expose the port your app listens on (change if different)
EXPOSE 8080

# Run the binary
CMD ["./app"]
