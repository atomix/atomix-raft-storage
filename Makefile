export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ifdef VERSION
ATOMIX_RAFT_STORAGE_VERSION := $(VERSION)
else
ATOMIX_RAFT_STORAGE_VERSION := latest
endif

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
	./build/bin/license-check

proto: # @HELP build Protobuf/gRPC generated types
proto:
	docker run -it -v `pwd`:/go/src/github.com/atomix/raft-storage \
		-w /go/src/github.com/atomix/raft-storage \
		--entrypoint build/bin/compile_protos.sh \
		onosproject/protoc-go:stable

images: # @HELP build atomix storage controller Docker images
images: build
	docker build . -f build/docker/Dockerfile -t atomix/raft-storage-controller:${ATOMIX_RAFT_STORAGE_VERSION}

kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image atomix/raft-storage-controller:${ATOMIX_RAFT_STORAGE_VERSION}

push: # @HELP push atomix-raft-node Docker image
	docker push atomix/raft-storage-controller:${ATOMIX_RAFT_STORAGE_VERSION}
