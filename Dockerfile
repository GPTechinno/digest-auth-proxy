FROM golang:alpine AS builder

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
# COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
RUN go build -o digest-auth-proxy .

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/digest-auth-proxy .

# Build a small image
FROM scratch

COPY --from=builder /dist/digest-auth-proxy /

# Export necessary port
EXPOSE 9999

# Export necessary env variable
ENV DAP_SERVER=""
ENV DAP_USER=""
ENV DAP_PASS=""

# Command to run
ENTRYPOINT ["/digest-auth-proxy"]