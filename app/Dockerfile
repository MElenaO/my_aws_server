############################
# STEP 1 build executable binary
############################

# Use the official Golang Alpine build image as the base image
FROM golang:alpine AS builder

# Create app directory in the image
RUN mkdir -p /app

# Copy the app code
COPY . /app

# Set the working directory inside the container
WORKDIR /app

# Build the application
RUN go build -o app

############################
# STEP 2 build a small image
############################
FROM alpine:latest

# Copy the executable
COPY --from=builder /app/app /app/app

WORKDIR /app

# Expose the port your application will listen on
EXPOSE 8080

# Run application
ENTRYPOINT ["./app"]