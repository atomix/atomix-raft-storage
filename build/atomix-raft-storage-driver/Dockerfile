FROM alpine:3.9

USER nobody

ADD build/_output/atomix-raft-storage-driver /usr/local/bin/atomix-raft-storage-driver

ENTRYPOINT ["atomix-raft-storage-driver"]
