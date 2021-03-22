package v2beta1

import (
	primitivesv2beta1 "github.com/atomix/kubernetes-controller/pkg/apis/primitives/v2beta1"
	"github.com/atomix/kubernetes-framework/pkg/controller"
	storagev2beta1 "github.com/atomix/raft-storage-controller/pkg/apis/storage/v2beta1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Add adds the primitive controller to the given manager
func Add(mgr manager.Manager) error {
	_, err := controller.NewBuilder().
		WithDriverImage("atomix/raft-storage-driver:latest").
		WithStorageType(&storagev2beta1.RaftProtocol{}).
		AddPrimitiveType(&primitivesv2beta1.Counter{}, &primitivesv2beta1.CounterList{}).
		AddPrimitiveType(&primitivesv2beta1.Election{}, &primitivesv2beta1.ElectionList{}).
		AddPrimitiveType(&primitivesv2beta1.List{}, &primitivesv2beta1.ListList{}).
		AddPrimitiveType(&primitivesv2beta1.Lock{}, &primitivesv2beta1.LockList{}).
		AddPrimitiveType(&primitivesv2beta1.Map{}, &primitivesv2beta1.MapList{}).
		AddPrimitiveType(&primitivesv2beta1.Set{}, &primitivesv2beta1.SetList{}).
		AddPrimitiveType(&primitivesv2beta1.Value{}, &primitivesv2beta1.ValueList{}).
		Build(mgr)
	return err
}
