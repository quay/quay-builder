.PHONY: vendor test bin/quay-builder

PROJECT   ?= quay-builder
ORG_PATH  ?= github.com/quay
REPO_PATH ?= $(ORG_PATH)/$(PROJECT)
IMAGE     ?= quay.io/projectquay/$(PROJECT)
VERSION   ?= $(shell ./scripts/git-version)
LD_FLAGS  ?= "-w -X $(REPO_PATH)/version.Version=$(VERSION)"
IMAGE_TAG ?= latest
SUBSCRIPTION_KEY ?= subscription.pem
BUILD_TAGS ?= 'btrfs_noversion exclude_graphdriver_btrfs exclude_graphdriver_devicemapper containers_image_openpgp'
BUILDER_SRC ?= 'github.com/quay/quay-builder'

all: vendor test build

vendor:
	@go mod vendor

test: vendor
	@go vet ./...
	@go test -v ./...

build: bin/quay-builder

bin/quay-builder:
	CGO_ENABLED=0 go build -ldflags $(LD_FLAGS) -o bin/quay-builder -tags $(BUILD_TAGS) $(REPO_PATH)/cmd/quay-builder

install:
	go install -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/quay-builder

build-centos:
	docker build --build-arg=BUILDER_SRC=$(BUILDER_SRC) -f Dockerfile.centos -t $(IMAGE):$(IMAGE_TAG)-centos .

build-alpine:
	docker build --build-arg=BUILDER_SRC=$(BUILDER_SRC) -f Dockerfile.alpine -t $(IMAGE):$(IMAGE_TAG)-alpine .
