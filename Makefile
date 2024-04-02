MODULE_NAME := $(shell go list -m)
DOCKER_IMAGE_NAME := ${shell basename ${MODULE_NAME}}-bookworm-slim
CWD := $(shell pwd)
BUILD_DIR := build
SCRAPE_PORT ?= 8080
TAG_VERSION ?= v0.0.0
TAG_MESSAGE ?= "Release version $(TAG_VERSION)"
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
	@go build -o $(BUILD_DIR)/ -tags "$(TAGS)" ./...
	
clean: ## clean the build directory
	@echo "Cleaning $(MODULE_NAME)..."
	@rm -rf $(BUILD_DIR)/*

docker-build: ## build a docker image on the current platform, for local use
	@echo "Building $(DOCKER_IMAGE_NAME)..." 
	@docker build -t $(DOCKER_IMAGE_NAME) .

docker-push: ## push an amd64/arm64 docker to Docker Hub or to a registry specified by CONTAINER_REGISTRY
	@echo "Pushing '$(PUSH_TAG)'..."
	@read -p "Do you want to push an amd64/arm64 image to '$(PUSH_TAG)'? (y/N) " answer; \
	if [ "$$answer" != "y" ]; then \
		echo "Aborted."; \
		exit 1; \
	fi
	@echo "Proceeding to make and push $(PUSH_TAG)..."
	@docker buildx create --use
	@docker buildx build --push --platform linux/amd64,linux/arm64 -t $(PUSH_TAG) . 

docker-run: ## run the local docker image, binding to port 8080, or the env value of SCRAPE_PORT
	@echo "Running $(DOCKER_IMAGE_NAME)..."
	@docker run -d -ti -p 127.0.0.1:$(SCRAPE_PORT):8080  --volume=$(CWD)/docker/data:/scrape_data --rm scrape-bookworm-slim:latest
 
fmt: 
	@echo "Running go fmt..."
	@go fmt ./...

release-tag: ## create a release tag with TAG_VERSION and TAG_MESSAGE
	@echo "Creating release tag $(TAG_VERSION) with message: $(TAG_MESSAGE)"
	@if [ "$(TAG_VERSION)" = "v0.0.0" ]; then \
        echo "Aborted. Release version cannot be 'v0.0.0'."; \
        exit 1; \
    fi
	@read -p "Continue to push this release tag? (y/n): " answer; \
    if [ "$$answer" != "y" ]; then \
        echo "Aborted."; \
        exit 1; \
    fi
	@git tag -a $(TAG_VERSION) -m $(TAG_MESSAGE)
	@git push origin $(TAG_VERSION)

test: ## run the tests
	@echo "Running tests..."
	@go test -coverprofile=coverage.out ./... 

test-mysql: ## run the MySQL integration tests
	@echo "Running MySQL tests..."
	@go test -tags mysql -coverprofile=mysql_coverage.out ./internal/storage ./internal/storage/mysql

vet: fmt ## fmt, vet, and staticcheck
	@echo "Running go vet and staticcheck..."
	@go vet ./...
	@staticcheck ./...

cognitive: ## run the cognitive complexity checker
	@echo "Running gocognit..."
	@gocognit  -ignore "_test|testdata" -top 5 .

help: ## show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


