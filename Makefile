.PHONY: build image push deploy undeploy test test-unit test-sanity test-e2e lint clean

# Variables
REGISTRY ?= ghcr.io
IMAGE_NAME ?= $(REGISTRY)/example/nfs-shared-csi
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
PLATFORMS ?= linux/amd64,linux/arm64

# Build binary
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/nfs-csi-driver ./cmd/nfs-csi-driver

# Build Docker image
image:
	docker build -t $(IMAGE_NAME):$(VERSION) .
	docker tag $(IMAGE_NAME):$(VERSION) $(IMAGE_NAME):latest

# Build multi-arch image
image-multiarch:
	docker buildx build --platform $(PLATFORMS) -t $(IMAGE_NAME):$(VERSION) --push .

# Push image
push:
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest

# Deploy to Kubernetes
deploy:
	kubectl apply -f deploy/kubernetes/

# Undeploy from Kubernetes
undeploy:
	kubectl delete -f deploy/kubernetes/ --ignore-not-found

# Run all tests (unit only, sanity requires root)
test: test-unit

# Run unit tests
test-unit:
	go test -v ./pkg/...

# Run CSI sanity tests (requires root for mount operations)
test-sanity:
	go test -v ./test/sanity/... -timeout 10m

# Run E2E tests (requires Kubernetes cluster and NFS server)
# Usage: NFS_SERVER=192.168.1.100 NFS_SHARE=/exports/data make test-e2e
test-e2e:
	go test -v ./test/e2e/... -timeout 30m

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean -cache

# Install dependencies
deps:
	go mod download
	go mod tidy

# Generate go.sum
tidy:
	go mod tidy

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build binary"
	@echo "  image          - Build Docker image"
	@echo "  image-multiarch - Build multi-arch Docker image"
	@echo "  push           - Push Docker image"
	@echo "  deploy         - Deploy to Kubernetes"
	@echo "  undeploy       - Undeploy from Kubernetes"
	@echo "  test           - Run unit tests"
	@echo "  test-unit      - Run unit tests"
	@echo "  test-sanity    - Run CSI sanity tests (requires root)"
	@echo "  test-e2e       - Run E2E tests (requires K8s + NFS)"
	@echo "  lint           - Run linter"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  tidy           - Run go mod tidy"
