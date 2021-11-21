ARG BASE_VERSION=latest
ARG ENGINE_VERSION=master

FROM atomix/atomix-go-base:${BASE_VERSION} AS build

COPY Makefile go.mod go.sum /build/
COPY cmd/ /build/cmd/
COPY pkg/ /build/pkg/
COPY vendor/ /build/vendor/

WORKDIR /build

RUN GOFLAGS=-mod=vendor ENGINE_DIR=/build/engine VERSION=${ENGINE_VERSION} make build-engine

FROM atomix/atomix-engine:${BASE_VERSION}

RUN apk upgrade --update --no-cache

USER nobody

RUN mkdir -p /etc/atomix/engines

COPY --from=build /build/engine/raft-${ENGINE_VERSION}.so /etc/atomix/engines/raft/raft-${ENGINE_VERSION}.so

ENTRYPOINT ["atomix-engine", "/etc/atomix/engines/raft/raft-${ENGINE_VERSION}.so"]
