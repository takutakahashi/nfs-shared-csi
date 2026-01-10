# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /nfs-csi-driver ./cmd/nfs-csi-driver

# Runtime stage
FROM alpine:3.19

# Install NFS utilities and util-linux (for GNU flock which supports -e option)
RUN apk add --no-cache nfs-utils util-linux

COPY --from=builder /nfs-csi-driver /nfs-csi-driver

ENTRYPOINT ["/nfs-csi-driver"]
