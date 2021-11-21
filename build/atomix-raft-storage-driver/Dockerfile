ARG BASE_VERSION=latest
ARG DRIVER_VERSION=master

FROM atomix/atomix-go-base:${BASE_VERSION} AS build

COPY Makefile go.mod go.sum /build/
COPY cmd/ /build/cmd/
COPY pkg/ /build/pkg/
COPY vendor/ /build/vendor/

WORKDIR /build

RUN GOFLAGS=-mod=vendor ENGINE_DIR=/build/driver VERSION=${DRIVER_VERSION} make build-driver

FROM atomix/atomix-driver:${BASE_VERSION}

RUN apk upgrade --update --no-cache

USER nobody

RUN mkdir -p /etc/atomix/drivers

COPY --from=build /build/driver/raft-${DRIVER_VERSION}.so /etc/atomix/drivers/raft/raft-${DRIVER_VERSION}.so

ENTRYPOINT ["atomix-driver", "/etc/atomix/drivers/raft/raft-${DRIVER_VERSION}.so"]
