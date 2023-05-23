# Use the official Golang image as the base image
FROM golang:1.16-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY go.mod .
COPY main.go .

# Build the Go application
RUN go build -o greet .

# Expose the port on which the service will run
EXPOSE 8080

# Set the command to run the executable when the container starts
CMD ["./greet"]

