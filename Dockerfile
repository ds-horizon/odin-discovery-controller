ARG GOVERSION=1.17
ARG GOOS=linux
ARG ALPINE_VERSION=3.18
# Use an official Go runtime as the base image
FROM golang:${GOVERSION} AS builder

ARG GOARCH
ENV GOARCH=${GOARCH}

# Set the working directory inside the container
WORKDIR /app
#
# Copy the Go modules files
COPY go.mod go.sum ./

# Download and cache Go modules
RUN go mod download

# Copy the the application code
COPY . .

# Build the Go application, disabling CGO to have self sufficient binaries by using static linking
RUN CGO_ENABLED=0 GOOS=${GOOS} go build -o odin-discovery-controller /app/cmd/main.go

# Create a minimal final image
FROM alpine:${ALPINE_VERSION}

# Set the working directory inside the container
WORKDIR /app

# Copy the built executable from the previous stage
COPY --from=builder /app/odin-discovery-controller .

## Command to run the controller
ENTRYPOINT ["./odin-discovery-controller"]
