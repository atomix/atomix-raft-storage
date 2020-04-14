export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ATOMIX_RAFT_STORAGE_VERSION := latest

all: build

build: # @HELP build the source code
build: deps
	GOOS=linux GOARCH=amd64 go build -o build/_output/raft-storage-controller ./cmd/raft-storage-controller


deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...

test: # @HELP run the unit tests and source code validation
test: build license_check linters
	go test github.com/atomix/raft-storage-controller/...

coverage: # @HELP generate unit test coverage data
coverage: build linters license_check


linters: # @HELP examines Go source code and reports coding problems
	GOGC=50 golangci-lint run

license_check: # @HELP examine and ensure license headers exist
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi
	./../build-tools/licensing/boilerplate.py -v --rootdir=${CURDIR}

proto: # @HELP build Protobuf/gRPC generated types
proto:
	docker run -it -v `pwd`:/go/src/github.com/atomix/raft-storage \
		-w /go/src/github.com/atomix/raft-storage \
		--entrypoint build/bin/compile_protos.sh \
		onosproject/protoc-go:stable

image: # @HELP build atomix storage controller Docker images
image: build
	docker build . -f build/docker/Dockerfile -t atomix/raft-storage-controller:${ATOMIX_RAFT_STORAGE_VERSION}

push: # @HELP push atomix-raft-node Docker image
	docker push atomix/raft-storage-controller:${ATOMIX_RAFT_STORAGE_VERSION}
