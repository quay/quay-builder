.PHONY: dep test bin/quay-builder

PROJECT   ?= quay-builder
ORG_PATH  ?= github.com/quay
REPO_PATH ?= $(ORG_PATH)/$(PROJECT)
IMAGE     ?= quay.io/projectquay/$(PROJECT)
VERSION   ?= $(shell ./scripts/git-version)
LD_FLAGS  ?= "-w -X $(REPO_PATH)/version.Version=$(VERSION)"
IMAGE_TAG ?= latest
SUBSCRIPTION_KEY ?= subscription.pem

all: dep test build

dep:
	@GO111MODULE=on go mod vendor

test: dep
	@go vet ./...
	@go test -v ./...

build: dep bin/quay-builder

bin/quay-builder:
	@go build -ldflags $(LD_FLAGS) -o bin/quay-builder \
	  $(REPO_PATH)/cmd/quay-builder

install:
	@go install -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/quay-builder

build-centos7:
	docker build -f Dockerfile.centos7 -t $(IMAGE):$(IMAGE_TAG)-centos7 .

build-rhel7:
	docker build -f Dockerfile.rhel7 -t $(IMAGE):$(IMAGE_TAG)-rhel7 . \
		--build-arg SUBSCRIPTION_KEY=$(SUBSCRIPTION_KEY)

build-alpine:
	docker build -f Dockerfile.alpine -t $(IMAGE):$(IMAGE_TAG)-alpine .
