FROM alpine:3.9

USER nobody

ADD build/_output/atomix-raft-storage-node /usr/local/bin/atomix-raft-storage-node

ENTRYPOINT ["atomix-raft-storage-node"]
