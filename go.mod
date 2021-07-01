module github.com/atomix/atomix-raft-storage

go 1.12

require (
	github.com/atomix/api v0.3.3
	github.com/atomix/atomix-api/go v0.4.5
	github.com/atomix/atomix-controller v0.5.0
	github.com/atomix/atomix-go-framework v0.8.1
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/lni/dragonboat/v3 v3.1.1-0.20201211124920-79d5e54396f7
	github.com/stretchr/testify v1.6.1
	google.golang.org/grpc v1.33.2
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	sigs.k8s.io/controller-runtime v0.5.2
)
