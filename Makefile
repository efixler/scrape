MODULE_NAME := $(shell go list -m)
DOCKER_IMAGE_NAME := ${shell basename ${MODULE_NAME}}-bookworm-slim
CWD := $(shell pwd)
BUILD_DIR := build
SCRAPE_PORT ?= 8080

.DEFAULT_GOAL := build

.PHONY: fmt vet build clean


build: vet ## build the binaries, to the build/ folder (post vet)
	@echo "Building $(MODULE_NAME)..."
	@go build -o $(BUILD_DIR)/ ./cmd/scrape/... 
	@go build -o $(BUILD_DIR)/ ./cmd/scrape-server/... 

clean: ## clean the build directory
	@echo "Cleaning $(MODULE_NAME)..."
	@rm -rf $(BUILD_DIR)/*

# Docker images pull code from the repo via `go install` so
# we skip the local checks here.
# So - changes to the source base won't show up these dockers
# until the hit the `latest` tag.
# TODO: Add a `docker-build-dev` target that builds a docker 
# image with the local source code.
docker-build: ## build the docker image
	@echo "Building $(DOCKER_IMAGE_NAME)..."
	@docker build --no-cache -t $(DOCKER_IMAGE_NAME) .

docker-run: ## run the docker image, binding to port 8080, or the env value of SCRAPE_PORT
	@echo "Running $(DOCKER_IMAGE_NAME)..."
	@docker run -d -ti -p 127.0.0.1:$(SCRAPE_PORT):8080  --volume=$(CWD)/docker/data:/scrape_data --rm scrape-bookworm-slim:latest
 
fmt: 
	@echo "Running go fmt..."
	@go fmt ./...
	@go fmt ./cmd/scrape/...
	@go fmt ./cmd/scrape-server/...

vet: fmt ## fmt, vet, and staticcheck
	@echo "Running go vet and staticcheck..."
	@go vet ./...
	@go vet ./cmd/scrape/...
	@go vet ./cmd/scrape-server/...
	@staticcheck ./...
	@staticcheck ./cmd/scrape/...
	@staticcheck ./cmd/scrape-server/...

help: ## show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

