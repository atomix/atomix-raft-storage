# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ifdef VERSION
MAJOR := $(word 1, $(subst ., , $(VERSION)))
MINOR := $(word 2, $(subst ., , $(VERSION)))
PATCH := $(word 3, $(subst ., , $(VERSION)))
endif

all: build

build: # @HELP build the source code
build: deps
	GOOS=linux GOARCH=amd64 go build -gcflags=-trimpath=${GOPATH} -asmflags=-trimpath=${GOPATH} -o build/_output/atomix-raft-storage-node ./cmd/atomix-raft-storage-node
	GOOS=linux GOARCH=amd64 go build -gcflags=-trimpath=${GOPATH} -asmflags=-trimpath=${GOPATH} -o build/_output/atomix-raft-storage-driver ./cmd/atomix-raft-storage-driver
	GOOS=linux GOARCH=amd64 go build -gcflags=-trimpath=${GOPATH} -asmflags=-trimpath=${GOPATH} -o build/_output/atomix-raft-storage-controller ./cmd/atomix-raft-storage-controller

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...

test: # @HELP run the unit tests and source code validation
test: build license linters
	go test github.com/atomix/atomix-raft-storage/...

coverage: # @HELP generate unit test coverage data
coverage: build license linters

linters: # @HELP examines Go source code and reports coding problems
	GOGC=50 golangci-lint run

reuse-tool: # @HELP install reuse if not present
	command -v reuse || python3 -m pip install reuse

license: reuse-tool # @HELP run license checks
	reuse lint

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
