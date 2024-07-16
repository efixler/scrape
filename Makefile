MODULE_NAME := scrape
ifneq ($(shell command -v go >/dev/null 2>&1 && echo yes),)
    MODULE_NAME := $(shell go list -m)
endif
DOCKER_IMAGE_NAME := ${shell basename ${MODULE_NAME}}-bookworm-slim
CWD := $(shell pwd)
BUILD_DIR := build
SCRAPE_PORT ?= 8080
CONTAINER_REGISTRY ?= docker.io


.DEFAULT_GOAL := build

.PHONY: fmt vet build clean test-mysql


build: vet ## build the binaries, to the build/ folder (default target)
	@echo "Building $(MODULE_NAME)..."
	@go build -o $(BUILD_DIR)/ -tags "$(TAGS)" ./...
	
clean: ## clean the build directory
	@echo "Cleaning $(MODULE_NAME)..."
	@rm -rf $(BUILD_DIR)/*

docker-build: ## build a docker image on the current platform, for local use
	@echo "Building $(DOCKER_IMAGE_NAME)..." 
	@docker build -t $(DOCKER_IMAGE_NAME) .

docker-push: push-tag ## push an amd64/arm64 docker to Docker Hub or to a registry specified by CONTAINER_REGISTRY
	@read -p "Do you want to push an amd64/arm64 image to '$(PUSH_TAG)' and '$(LATEST_TAG)'? (y/N) " answer; \
	if [ "$$answer" != "y" ]; then \
		echo "Aborted."; \
		exit 1; \
	fi
	@echo "Proceeding to make and push $(PUSH_TAG)..."
	@docker buildx create --use
	@docker buildx build --push --platform linux/amd64,linux/arm64 -t $(PUSH_TAG) -t $(LATEST_TAG) . 

push-tag: latest-release-tag 
	$(eval GITHUB_ORG := $(shell git remote get-url origin | sed -n 's/.*github.com[:/]\([^/]*\)\/.*/\1/p' | tr '[:upper:]' '[:lower:]'))
	$(eval PUSH_TAG=$(CONTAINER_REGISTRY)/$(GITHUB_ORG)/$(DOCKER_IMAGE_NAME):$(RELEASE_TAG))
	$(eval LATEST_TAG=$(CONTAINER_REGISTRY)/$(GITHUB_ORG)/$(DOCKER_IMAGE_NAME):latest)

docker-run: ## run the local docker image, binding to port 8080, or the env value of SCRAPE_PORT
	@echo "Running $(DOCKER_IMAGE_NAME)..."
	@docker run -d -ti -p 127.0.0.1:$(SCRAPE_PORT):8080  --volume=$(CWD)/docker/data:/scrape_data --rm scrape-bookworm-slim:latest
 
fmt: 
	@echo "Running go fmt..."
	@go fmt ./...

release-tag: latest-release-tag ## create a release tag at the next patch version. Customize with TAG_MESSAGE and/or TAG_VERSION
	$(eval TAG_VERSION ?= $(shell echo $(RELEASE_TAG) | awk -F. '{print $$1"."$$2"."$$3+1}'))
	$(eval TAG_MESSAGE ?= "Release version $(TAG_VERSION)")
	@echo "Creating release tag $(TAG_VERSION) with message: \"$(TAG_MESSAGE)\""
	@if [ "$(TAG_VERSION)" = "v0.0.0" ]; then \
        echo "Aborted. Release version cannot be 'v0.0.0'."; \
        exit 1; \
    fi
	@read -p "Continue to push this release tag? (y/n): " answer; \
    if [ "$$answer" != "y" ]; then \
        echo "Aborted."; \
        exit 1; \
    fi
	@git tag -a $(TAG_VERSION) -m '$(TAG_MESSAGE)'
	@git push origin $(TAG_VERSION)

latest-release-tag: 
	$(eval RELEASE_TAG := $(shell git describe --abbrev=0 --tags $(shell git rev-list --tags --max-count=1) 2>/dev/null || echo "v0.0.0"))
	@echo "Latest release tag: $(RELEASE_TAG)"

test: ## run the tests
	@echo "Running tests..."
	@go test -coverprofile=coverage.out ./... 

test-mysql: ## run the MySQL integration tests
	@echo "Running MySQL tests..."
	@go test -tags mysql -coverprofile=mysql_coverage.out ./internal/storage ./database/mysql

vet: fmt ## fmt, vet, and staticcheck
	@echo "Running go vet and staticcheck..."
	@go vet ./...
	@staticcheck ./...

cognitive: ## run the cognitive complexity checker
	@echo "Running gocognit..."
	@gocognit  -ignore "_test|testdata" -top 5 .

help: ## show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


