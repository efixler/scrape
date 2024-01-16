MODULE_NAME := $(shell go list -m)
DOCKER_IMAGE_NAME := ${shell basename ${MODULE_NAME}}-bookworm-slim
CWD := $(shell pwd)
BUILD_DIR := build
SCRAPE_PORT ?= 8080
define GITHUB_ORG_USER
$(shell git remote get-url origin | sed -n 's/.*github.com[:/]\([^/]*\)\/.*/\1/p' | tr '[:upper:]' '[:lower:]')
endef
CONTAINER_REGISTRY ?= docker.io
define PUSH_TAG
$(CONTAINER_REGISTRY)/$(GITHUB_ORG_USER)/$(DOCKER_IMAGE_NAME):latest
endef

.DEFAULT_GOAL := build

.PHONY: fmt vet build clean


build: vet ## build the binaries, to the build/ folder (default target)
	@echo "Building $(MODULE_NAME)..."
	@go build -o $(BUILD_DIR)/ -tags "$(TAGS)" ./cmd/scrape/... 
	@go build -o $(BUILD_DIR)/ -tags "$(TAGS)" ./cmd/scrape-feed/... 
	@go build -o $(BUILD_DIR)/ -tags "$(TAGS)" ./cmd/scrape-server/... 

clean: ## clean the build directory
	@echo "Cleaning $(MODULE_NAME)..."
	@rm -rf $(BUILD_DIR)/*

# Docker images pull code from the repo via `go install` so
# we skip the local checks here.
# So - changes to the source base won't show up these dockers
# until they hit the `latest` tag.
# TODO: Add a `docker-build-dev` target that builds a docker 
# image with the local source code.
docker-build: ## build a docker image on the current platform, for local use
	@echo "Building $(DOCKER_IMAGE_NAME)..." 
	@docker build --no-cache -t $(DOCKER_IMAGE_NAME) .

docker-push: ## push an amd64/arm64 docker to Docker Hub or to a registry specfied by CONTAINER_REGISTRY
	@echo "Pushing '$(PUSH_TAG)'..."
	@read -p "Do you want to push an amd64/arm64 image to '$(PUSH_TAG)'? (y/N) " answer; \
	if [ "$$answer" != "y" ]; then \
		echo "Aborted."; \
		exit 1; \
	fi
	@echo "Proceeding to make and push $(PUSH_TAG)..."
	@docker buildx create --use
	@docker buildx build --push --platform linux/amd64,linux/arm64 --no-cache -t $(PUSH_TAG) . 

docker-run: ## run the local docker image, binding to port 8080, or the env value of SCRAPE_PORT
	@echo "Running $(DOCKER_IMAGE_NAME)..."
	@docker run -d -ti -p 127.0.0.1:$(SCRAPE_PORT):8080  --volume=$(CWD)/docker/data:/scrape_data --rm scrape-bookworm-slim:latest
 
fmt: 
	@echo "Running go fmt..."
	@go fmt ./...
	@go fmt ./cmd/scrape/...
	@go fmt ./cmd/scrape-feed/...
	@go fmt ./cmd/scrape-server/...

test: ## run the tests
	@echo "Running tests..."
	@go test -coverprofile=coverage.out ./... 

vet: fmt ## fmt, vet, and staticcheck
	@echo "Running go vet and staticcheck..."
	@go vet ./...
	@go vet ./cmd/scrape/...
	@go vet ./cmd/scrape-feed/...
	@go vet ./cmd/scrape-server/...
	@staticcheck ./...
	@staticcheck ./cmd/scrape/...
	@staticcheck ./cmd/scrape-feed/...
	@staticcheck ./cmd/scrape-server/...

cognitive: ## run the cognitive complexity checker
	@echo "Running gocognit..."
	@gocognit  -ignore "_test|testdata" -top 5 .

help: ## show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


