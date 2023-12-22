MODULE_NAME := $(shell go list -m)
DOCKER_IMAGE_NAME := ${shell basename ${MODULE_NAME}}-bookworm-slim
CWD := $(shell pwd)

.DEFAULT_GOAL := build

.PHONY: fmt vet build clean

fmt:
	@echo "Running go fmt..."
	@go fmt ./...
	@go fmt ./cmd/scrape/...
	@go fmt ./cmd/scrape-server/...

vet: fmt
	@echo "Running go vet and staticcheck..."
	@go vet ./...
	@go vet ./cmd/scrape/...
	@go vet ./cmd/scrape-server/...
	@staticcheck ./...
	@staticcheck ./cmd/scrape/...
	@staticcheck ./cmd/scrape-server/...

build: vet
	@echo "Building $(MODULE_NAME)..."
	@go build -o bin/ ./cmd/scrape/... 
	@go build -o bin/ ./cmd/scrape-server/... 

build-docker: 
	@echo "Building $(DOCKER_IMAGE_NAME)..."
	@docker build --no-cache -t $(DOCKER_IMAGE_NAME) .

run-docker: 
	@echo "Running $(DOCKER_IMAGE_NAME)..."
	@docker run -dp 8080:8080  --volume=$(CWD)/docker/data:/scrape_data --volume=/scrape_data --rm -ti scrape-bookworm-slim:latest

clean:
	@echo "Cleaning $(MODULE_NAME)..."
	@rm -rf bin/*