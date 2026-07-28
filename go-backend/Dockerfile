# Dockerfile-backend

# Use the official Golang image as a parent image
FROM golang:1.25

# Set the working directory in the container
WORKDIR /usr/src/app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o main .

# Make port 8080 available to the world outside this container
EXPOSE 8080

# Run the Go app when the container launches
CMD ["./main"]
