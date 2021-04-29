module github.com/atomix/atomix-raft-storage-plugin

go 1.12

require (
	github.com/atomix/api v0.3.3
	github.com/atomix/atomix-api/go v0.3.3
	github.com/atomix/atomix-go-framework v0.5.1
	github.com/atomix/atomix-controller v0.4.0
	github.com/gogo/protobuf v1.3.1
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.2
)

replace github.com/atomix/atomix-api/go => ../atomix-api/go

replace github.com/atomix/atomix-go-framework => ../atomix-go-node

replace github.com/atomix/atomix-controller => ../atomix-k8s-controller
