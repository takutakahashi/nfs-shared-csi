# Build stage
FROM golang:1.22-alpine AS builder

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

RUN apk add --no-cache nfs-utils

COPY --from=builder /nfs-csi-driver /nfs-csi-driver

ENTRYPOINT ["/nfs-csi-driver"]
