# Use official Golang image
FROM golang:1.21.5-alpine

# Set working sirectoryu
WORKDIR /app

# Copy the source code
COPY . .

# Download and install the dependencies
RUN go mod tidy && go mod download

# Build the Go app
RUN go build -o api .

# Expose the port
EXPOSE 8000

# Run the executable
CMD ["./api"]