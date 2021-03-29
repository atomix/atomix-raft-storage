package v2beta1

import (
	"context"
	"fmt"
	"github.com/atomix/api/go/atomix/management/broker"
	protocolapi "github.com/atomix/api/go/atomix/protocol"
	primitivesv2beta1 "github.com/atomix/kubernetes-controller/pkg/apis/primitives/v2beta1"
	ctrlv2beta1 "github.com/atomix/kubernetes-controller/pkg/controller/primitives/v2beta1"
	storagev2beta1 "github.com/atomix/raft-storage-controller/pkg/apis/storage/v2beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	driverTypeEnv      = "ATOMIX_DRIVER_TYPE"
	driverNodeEnv      = "ATOMIX_DRIVER_NODE"
	driverNamespaceEnv = "ATOMIX_DRIVER_NAMESPACE"
	driverNameEnv      = "ATOMIX_DRIVER_NAME"
	driverPortEnv      = "ATOMIX_DRIVER_PORT"
	clusterDomainEnv   = "CLUSTER_DOMAIN"
)

const (
	apiPort          = 5678
	protocolPortName = "raft"
	protocolPort     = 5679
)

const headlessServiceSuffix = "hs"

// AddControllers adds the driver controller to the given manager
func AddControllers(mgr manager.Manager) error {
	_, err := ctrlv2beta1.NewControllerBuilder("raft").
		WithStorageController(newDriverController(mgr)).
		WithResourceType(&storagev2beta1.RaftProtocol{}).
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

func newDriverController(mgr manager.Manager) ctrlv2beta1.StorageController {
	return &Controller{
		client: mgr.GetClient(),
	}
}

type Controller struct {
	client client.Client
}

func (c *Controller) GetProtocol(driver broker.DriverId) (protocolapi.ProtocolConfig, error) {
	protocol := &storagev2beta1.RaftProtocol{}
	protocolName := types.NamespacedName{
		Namespace: driver.Namespace,
		Name:      driver.Name,
	}
	if err := c.client.Get(context.TODO(), protocolName, protocol); err != nil {
		return protocolapi.ProtocolConfig{}, err
	}

	numClusters := getClusters(protocol)
	numReplicas := getReplicas(protocol)
	replicas := make([]protocolapi.ProtocolReplica, 0, numReplicas*numClusters)
	for i := 0; i < numClusters; i++ {
		for j := 0; j < numReplicas; j++ {
			replica := protocolapi.ProtocolReplica{
				ID:      getPodName(protocol, j, i),
				Host:    getPodDNSName(protocol, j, i),
				APIPort: apiPort,
				ExtraPorts: map[string]int32{
					protocolPortName: protocolPort,
				},
			}
			replicas = append(replicas, replica)
		}
	}

	partitions := make([]protocolapi.ProtocolPartition, 0, protocol.Spec.Partitions)
	for partitionID := 1; partitionID <= int(protocol.Spec.Partitions); partitionID++ {
		for i := 0; i < numClusters; i++ {
			replicaNames := make([]string, 0, numReplicas)
			for j := 0; j < numReplicas; j++ {
				replicaNames = append(replicaNames, getPodName(protocol, j, i))
			}
			partition := protocolapi.ProtocolPartition{
				PartitionID: uint32(partitionID),
				Replicas:    replicaNames,
			}
			partitions = append(partitions, partition)
		}
	}

	return protocolapi.ProtocolConfig{
		Replicas:   replicas,
		Partitions: partitions,
	}, nil
}

func (c *Controller) NewContainer(driver broker.DriverConfig) (corev1.Container, error) {
	return corev1.Container{
		Name:            fmt.Sprintf("atomix-%s-driver", driver.ID.Name),
		Image:           "atomix/dragonboat-raft-storage-driver:latest",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name:  driverTypeEnv,
				Value: driver.ID.Type,
			},
			{
				Name:  driverNamespaceEnv,
				Value: driver.ID.Namespace,
			},
			{
				Name:  driverNameEnv,
				Value: driver.ID.Name,
			},
			{
				Name:  driverPortEnv,
				Value: fmt.Sprint(driver.Port),
			},
			{
				Name: driverNodeEnv,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
		},
	}, nil
}

func (c *Controller) NewEphemeralContainer(driver broker.DriverConfig) (corev1.EphemeralContainer, error) {
	return corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            fmt.Sprintf("atomix-%s-driver", driver.ID.Name),
			Image:           "atomix/dragonboat-raft-storage-driver:latest",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: []corev1.EnvVar{
				{
					Name:  driverTypeEnv,
					Value: driver.ID.Type,
				},
				{
					Name:  driverNamespaceEnv,
					Value: driver.ID.Namespace,
				},
				{
					Name:  driverNameEnv,
					Value: driver.ID.Name,
				},
				{
					Name:  driverPortEnv,
					Value: fmt.Sprint(driver.Port),
				},
				{
					Name: driverNodeEnv,
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				},
			},
		},
	}, nil
}

// getClusterName returns the cluster name
func getClusterName(storage *storagev2beta1.RaftProtocol, cluster int) string {
	return fmt.Sprintf("%s-%d", storage.Name, cluster)
}

// getPodName returns the name of the pod for the given pod ID
func getPodName(storage *storagev2beta1.RaftProtocol, cluster int, pod int) string {
	return fmt.Sprintf("%s-%d", getClusterName(storage, cluster), pod)
}

// getPodDNSName returns the fully qualified DNS name for the given pod ID
func getPodDNSName(storage *storagev2beta1.RaftProtocol, cluster int, pod int) string {
	domain := os.Getenv(clusterDomainEnv)
	if domain == "" {
		domain = "cluster.local"
	}
	return fmt.Sprintf("%s-%d.%s.%s.svc.%s", getClusterName(storage, cluster), pod, getClusterHeadlessServiceName(storage, cluster), storage.Namespace, domain)
}

// getClusterHeadlessServiceName returns the headless service name for the given cluster
func getClusterHeadlessServiceName(storage *storagev2beta1.RaftProtocol, cluster int) string {
	return getClusterResourceName(storage, cluster, headlessServiceSuffix)
}

// getClusterResourceName returns the given resource name for the given cluster
func getClusterResourceName(storage *storagev2beta1.RaftProtocol, cluster int, resource string) string {
	return fmt.Sprintf("%s-%s", getClusterName(storage, cluster), resource)
}

// getClusterForPartitionID returns the cluster ID for the given partition ID
func getClusterForPartitionID(storage *storagev2beta1.RaftProtocol, partition int) int {
	return (partition % getClusters(storage)) + 1
}

// getClusters returns the number of clusters in the given database
func getClusters(storage *storagev2beta1.RaftProtocol) int {
	if storage.Spec.Clusters == 0 {
		return 1
	}
	return int(storage.Spec.Clusters)
}

// getReplicas returns the number of replicas in the given database
func getReplicas(storage *storagev2beta1.RaftProtocol) int {
	if storage.Spec.Replicas == 0 {
		return 1
	}
	return int(storage.Spec.Replicas)
}
