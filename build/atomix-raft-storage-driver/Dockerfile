# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

FROM alpine:3.9

USER nobody

ADD build/_output/atomix-raft-storage-driver /usr/local/bin/atomix-raft-storage-driver

ENTRYPOINT ["atomix-raft-storage-driver"]
