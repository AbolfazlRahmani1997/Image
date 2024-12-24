FROM golang:1.23

# Set working directory
WORKDIR /app

# Copy Go modules and install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application code
COPY . .

# Build the Go application
RUN go build -o main ./cmd

# Expose the port the service runs on
EXPOSE 8080

# Command to run the application
CMD ["./main"]