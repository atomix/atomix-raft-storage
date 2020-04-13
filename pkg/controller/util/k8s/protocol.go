// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8s

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	api "github.com/atomix/api/proto/atomix/controller"
	"github.com/atomix/kubernetes-controller/pkg/apis/cloud/v1beta2"
	storage "github.com/atomix/raft-storage-controller/pkg/apis/v1beta1"
	"github.com/gogo/protobuf/jsonpb"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	apiPort            = 5678
	protocolPort       = 5679
	probePort          = 5679
	databaseAnnotation = "cloud.atomix.io/database"
	clusterAnnotation  = "cloud.atomix.io/cluster"
)

// NewClusterConfigMap returns a new ConfigMap for initializing Atomix clusters
func NewClusterConfigMap(cluster *v1beta2.Cluster, storage *storage.RaftStorageClass, config interface{}) (*corev1.ConfigMap, error) {
	clusterConfig, err := newNodeConfigString(cluster, storage)
	if err != nil {
		return nil, err
	}

	protocolConfig, err := newProtocolConfigString(config)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
		},
		Data: map[string]string{
			clusterConfigFile:  clusterConfig,
			protocolConfigFile: protocolConfig,
		},
	}, nil
}

// getPodName returns the name of the pod for the given pod ID
func getPodName(cluster *v1beta2.Cluster, pod int) string {
	return fmt.Sprintf("%s-%d", cluster.Name, pod)
}

// getPodDNSName returns the fully qualified DNS name for the given pod ID
func getPodDNSName(cluster *v1beta2.Cluster, pod int) string {
	return fmt.Sprintf("%s-%d.%s.%s.svc.cluster.local", cluster.Name, pod, cluster.Name, cluster.Namespace)
}

// GetDatabaseFromClusterAnnotations returns the database name from the given cluster annotations
func GetDatabaseFromClusterAnnotations(cluster *v1beta2.Cluster) (string, error) {
	database, ok := cluster.Annotations[databaseAnnotation]
	if !ok {
		return "", errors.New("cluster missing database annotation")
	}
	return database, nil
}

// GetClusterIDFromClusterAnnotations returns the cluster ID from the given cluster annotations
func GetClusterIDFromClusterAnnotations(cluster *v1beta2.Cluster) (int32, error) {
	idstr, ok := cluster.Annotations[clusterAnnotation]
	if !ok {
		return 1, nil
	}

	id, err := strconv.ParseInt(idstr, 0, 32)
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}

// newNodeConfigString creates a node configuration string for the given cluster
func newNodeConfigString(cluster *v1beta2.Cluster, storage *storage.RaftStorageClass) (string, error) {
	clusterID, err := GetClusterIDFromClusterAnnotations(cluster)
	if err != nil {
		return "", err
	}

	clusterDatabase, err := GetDatabaseFromClusterAnnotations(cluster)
	if err != nil {
		return "", err
	}

	members := make([]*api.MemberConfig, storage.Spec.Replicas)
	for i := 0; i < int(storage.Spec.Replicas); i++ {
		members[i] = &api.MemberConfig{
			ID:           getPodName(cluster, i),
			Host:         getPodDNSName(cluster, i),
			ProtocolPort: protocolPort,
			APIPort:      apiPort,
		}
	}

	partitions := make([]*api.PartitionId, 0, cluster.Spec.Partitions)
	for partitionID := (cluster.Spec.Partitions * (clusterID - 1)) + 1; partitionID <= cluster.Spec.Partitions*clusterID; partitionID++ {
		partition := &api.PartitionId{
			Partition: partitionID,
			Cluster: &api.ClusterId{
				ID: int32(clusterID),
				DatabaseID: &api.DatabaseId{
					Name:      clusterDatabase,
					Namespace: cluster.Namespace,
				},
			},
		}
		partitions = append(partitions, partition)
	}

	config := &api.ClusterConfig{
		Members:    members,
		Partitions: partitions,
	}

	marshaller := jsonpb.Marshaler{}
	return marshaller.MarshalToString(config)
}

// newProtocolConfigString creates a protocol configuration string for the given cluster and protocol
func newProtocolConfigString(config interface{}) (string, error) {
	bytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// NewClusterDisruptionBudget returns a new pod disruption budget for the cluster group cluster
func NewClusterDisruptionBudget(cluster *v1beta2.Cluster, storage *storage.RaftStorageClass) *policyv1beta1.PodDisruptionBudget {
	minAvailable := intstr.FromInt(int(storage.Spec.Replicas)/2 + 1)
	return &policyv1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
		},
	}
}

// NewClusterHeadlessService returns a new headless service for a cluster group
func NewClusterHeadlessService(cluster *v1beta2.Cluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
			Annotations: map[string]string{
				"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "api",
					Port: apiPort,
				},
				{
					Name: "protocol",
					Port: protocolPort,
				},
			},
			PublishNotReadyAddresses: true,
			ClusterIP:                "None",
			Selector:                 cluster.Labels,
		},
	}
}

// NewStatefulSet returns a new StatefulSet for a cluster group
func NewStatefulSet(cluster *v1beta2.Cluster, storage *storage.RaftStorageClass) (*appsv1.StatefulSet, error) {
	volumes := []corev1.Volume{
		newConfigVolume(cluster.Name),
	}

	args := []string{
		"$(NODE_ID)",
		fmt.Sprintf("%s/%s", configPath, clusterConfigFile),
		fmt.Sprintf("%s/%s", configPath, protocolConfigFile),
	}

	volumes = append(volumes, newDataVolume())

	image := storage.Spec.Image
	pullPolicy := storage.Spec.ImagePullPolicy
	readinessProbe :=
		&corev1.Probe{
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"stat", "/tmp/atomix-ready"},
				},
			},
			InitialDelaySeconds: 5,
			TimeoutSeconds:      10,
			FailureThreshold:    12,
		}
	livenessProbe :=
		&corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.IntOrString{Type: intstr.Int, IntVal: probePort},
				},
			},
			InitialDelaySeconds: 60,
			TimeoutSeconds:      10,
		}

	if pullPolicy == "" {
		pullPolicy = corev1.PullIfNotPresent
	}

	apiContainerPort := corev1.ContainerPort{
		Name:          "api",
		ContainerPort: 5678,
	}
	protocolContainerPort := corev1.ContainerPort{
		Name:          "protocol",
		ContainerPort: 5679,
	}

	containerBuilder := NewContainer()
	container := containerBuilder.SetImage(image).
		SetName(cluster.Name).
		SetPullPolicy(pullPolicy).
		SetArgs(args...).
		SetPorts([]corev1.ContainerPort{apiContainerPort, protocolContainerPort}).
		SetReadinessProbe(readinessProbe).
		SetLivenessProbe(livenessProbe).
		SetVolumeMounts([]corev1.VolumeMount{newDataVolumeMount(), newConfigVolumeMount()}).
		Build()

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: cluster.Name,
			Replicas:    &storage.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: cluster.Labels,
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: cluster.Labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						newContainer(container),
					},
					Volumes: volumes,
				},
			},
		},
	}, nil
}
