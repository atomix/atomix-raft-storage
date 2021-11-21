module github.com/atomix/atomix-raft-storage

go 1.12

require (
	github.com/atomix/atomix-api/go v0.7.0
	github.com/atomix/atomix-controller v0.6.1
	github.com/atomix/atomix-go-framework v0.9.10 // indirect
	github.com/atomix/atomix-sdk-go v0.0.0-20211103073703-8ea4bd1484be
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/lni/dragonboat/v3 v3.3.4
	google.golang.org/grpc v1.38.0
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	sigs.k8s.io/controller-runtime v0.5.2
)

replace github.com/atomix/atomix-sdk-go => ../atomix-go-sdk
