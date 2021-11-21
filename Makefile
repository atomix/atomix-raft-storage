export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ifdef VERSION
MAJOR := $(word 1, $(subst ., , $(VERSION)))
MINOR := $(word 2, $(subst ., , $(VERSION)))
PATCH := $(word 3, $(subst ., , $(VERSION)))
DRIVER_FILE := raft-$(VERSION).so
ENGINE_FILE := raft-$(VERSION).so
else
DRIVER_FILE := raft-$(shell git rev-parse HEAD).so
ENGINE_FILE := raft-$(shell git rev-parse HEAD).so
endif

DRIVER_DIR ?= driver
ENGINE_DIR ?= engine

all: build

build-driver:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-gcflags=-trimpath=${GOPATH} \
		-asmflags=-trimpath=${GOPATH} \
		-buildmode=plugin \
		-o $(DRIVER_DIR)/$(DRIVER_FILE) \
		github.com/atomix/atomix-raft-storage/plugins/driver

build-engine:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-gcflags=-trimpath=${GOPATH} \
		-asmflags=-trimpath=${GOPATH} \
		-buildmode=plugin \
		-o $(ENGINE_DIR)/$(ENGINE_FILE) \
		github.com/atomix/atomix-raft-storage/plugins/engine

build: deps build-driver build-engine

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...

test: # @HELP run the unit tests and source code validation
test: build license_check linters
	go test github.com/atomix/atomix-raft-storage/...

coverage: # @HELP generate unit test coverage data
coverage: build linters license_check

linters: # @HELP examines Go source code and reports coding problems
	GOGC=50 golangci-lint run

license_check: # @HELP examine and ensure license headers exist
	./build/bin/license-check

proto: # @HELP build Protobuf/gRPC generated types
proto:
	docker run -it -v `pwd`:/go/src/github.com/atomix/atomix-raft-storage \
		-w /go/src/github.com/atomix/atomix-raft-storage \
		--entrypoint build/bin/compile_protos.sh \
		onosproject/protoc-go:stable

images: # @HELP build atomix storage controller Docker images
images: build
	docker build . -f build/atomix-raft-storage-node/Dockerfile       -t atomix/atomix-raft-storage-node:latest
	docker build . -f build/atomix-raft-storage-driver/Dockerfile     -t atomix/atomix-raft-storage-driver:latest
	docker build . -f build/atomix-raft-storage-controller/Dockerfile -t atomix/atomix-raft-storage-controller:latest
ifdef VERSION
	docker tag atomix/atomix-raft-storage-node:latest atomix/atomix-raft-storage-node:${MAJOR}.${MINOR}.${PATCH}
	docker tag atomix/atomix-raft-storage-node:latest atomix/atomix-raft-storage-node:${MAJOR}.${MINOR}
	docker tag atomix/atomix-raft-storage-driver:latest atomix/atomix-raft-storage-driver:${MAJOR}.${MINOR}.${PATCH}
	docker tag atomix/atomix-raft-storage-driver:latest atomix/atomix-raft-storage-driver:${MAJOR}.${MINOR}
	docker tag atomix/atomix-raft-storage-controller:latest atomix/atomix-raft-storage-controller:${MAJOR}.${MINOR}.${PATCH}
	docker tag atomix/atomix-raft-storage-controller:latest atomix/atomix-raft-storage-controller:${MAJOR}.${MINOR}
endif

kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image atomix/atomix-raft-storage-node:latest
	kind load docker-image atomix/atomix-raft-storage-driver:latest
	kind load docker-image atomix/atomix-raft-storage-controller:latest
ifdef VERSION
	kind load docker-image atomix/atomix-raft-storage-node:${MAJOR}.${MINOR}.${PATCH}
	kind load docker-image atomix/atomix-raft-storage-node:${MAJOR}.${MINOR}
	kind load docker-image atomix/atomix-raft-storage-driver:${MAJOR}.${MINOR}.${PATCH}
	kind load docker-image atomix/atomix-raft-storage-driver:${MAJOR}.${MINOR}
	kind load docker-image atomix/atomix-raft-storage-controller:${MAJOR}.${MINOR}.${PATCH}
	kind load docker-image atomix/atomix-raft-storage-controller:${MAJOR}.${MINOR}
endif

push: # @HELP push atomix-raft-node Docker image
	docker push atomix/atomix-raft-storage-node:latest
	docker push atomix/atomix-raft-storage-driver:latest
	docker push atomix/atomix-raft-storage-controller:latest
ifdef VERSION
	docker push atomix/atomix-raft-storage-node:${MAJOR}.${MINOR}.${PATCH}
	docker push atomix/atomix-raft-storage-node:${MAJOR}.${MINOR}
	docker push atomix/atomix-raft-storage-driver:${MAJOR}.${MINOR}.${PATCH}
	docker push atomix/atomix-raft-storage-driver:${MAJOR}.${MINOR}
	docker push atomix/atomix-raft-storage-controller:${MAJOR}.${MINOR}.${PATCH}
	docker push atomix/atomix-raft-storage-controller:${MAJOR}.${MINOR}
endif
