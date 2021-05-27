FROM alpine:3.9

USER nobody

ADD build/_output/atomix-raft-storage-controller /usr/local/bin/atomix-raft-storage-controller

ENTRYPOINT ["atomix-raft-storage-controller"]
