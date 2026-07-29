# Dockerfile-backend

# Use the official Golang image as a parent image
FROM golang:1.25

# Set the working directory in the container
WORKDIR /usr/src/app

# Default local file-storage root. Lands on the ./data:/usr/src/app/data
# volume mounted in docker-zettel-run.yml. Overridable at runtime via env_file
# / -e (see docs/plans/2026-07-29-s3-to-local-file-storage-design.md D7).
ENV STORAGE_DIR=/usr/src/app/data/files

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
