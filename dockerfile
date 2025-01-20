# Use the official Golang image as the base image
FROM golang:1.20-alpine

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod file and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o projeto-app .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./projeto-app"]
