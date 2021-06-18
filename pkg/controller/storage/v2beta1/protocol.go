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

package v2beta1

import (
	"context"
	"encoding/json"
	"fmt"
	protocolapi "github.com/atomix/atomix-api/go/atomix/protocol"
	storagev2beta1 "github.com/atomix/atomix-raft-storage/pkg/apis/storage/v2beta1"
	"github.com/atomix/atomix-raft-storage/pkg/storage"
	"github.com/gogo/protobuf/jsonpb"
	"google.golang.org/grpc"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	apiPort               = 5678
	protocolPortName      = "raft"
	protocolPort          = 5679
	probePort             = 5679
	defaultImageEnv       = "DEFAULT_NODE_V2BETA1_IMAGE"
	defaultImage          = "atomix/atomix-raft-storage-node:latest"
	headlessServiceSuffix = "hs"
	appLabel              = "app"
	databaseLabel         = "database"
	clusterLabel          = "cluster"
	appAtomix             = "atomix"
)

const (
	configPath         = "/etc/atomix"
	clusterConfigFile  = "cluster.json"
	protocolConfigFile = "protocol.json"
	dataPath           = "/var/lib/atomix"
)

const (
	configVolume = "config"
	dataVolume   = "data"
)

const monitoringPort = 5000

const clusterDomainEnv = "CLUSTER_DOMAIN"

func (r *Reconciler) reconcileClusters(protocol *storagev2beta1.MultiRaftProtocol) error {
	for _, clusterID := range getClusters(protocol) {
		err := r.reconcileCluster(protocol, clusterID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) reconcileCluster(protocol *storagev2beta1.MultiRaftProtocol, clusterID int) error {
	log.Info("Reconcile raft protocol cluster")
	cluster := &storagev2beta1.RaftCluster{}
	name := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getClusterName(protocol, clusterID),
	}
	err := r.client.Get(context.TODO(), name, cluster)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		err = r.addCluster(protocol, clusterID)
		if err != nil {
			return err
		}
		err = r.client.Get(context.TODO(), name, cluster)
		if err != nil {
			return err
		}
	}

	err = r.reconcilePartitions(protocol, cluster)
	if err != nil {
		return err
	}

	err = r.reconcileConfigMap(protocol, cluster)
	if err != nil {
		return err
	}

	err = r.reconcileStatefulSet(protocol, cluster)
	if err != nil {
		return err
	}

	err = r.reconcileService(protocol, cluster)
	if err != nil {
		return err
	}

	// TODO: Stop monitoring clusters too!
	err = r.startMonitoringCluster(protocol, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) addCluster(protocol *storagev2beta1.MultiRaftProtocol, clusterID int) error {
	cluster := &storagev2beta1.RaftCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: protocol.Namespace,
			Name:      getClusterName(protocol, clusterID),
		},
		Spec: storagev2beta1.RaftClusterSpec{
			ClusterID: int32(clusterID),
		},
	}
	if err := controllerutil.SetControllerReference(protocol, cluster, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), cluster)
}

func (r *Reconciler) reconcilePartitions(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) error {
	for _, partitionID := range getPartitions(protocol, int(cluster.Spec.ClusterID)) {
		err := r.reconcilePartition(protocol, cluster, partitionID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) reconcilePartition(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster, partitionID int) error {
	log.Info("Reconcile raft protocol partition")
	partition := &storagev2beta1.RaftPartition{}
	name := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getPartitionName(protocol, int(cluster.Spec.ClusterID), partitionID),
	}
	err := r.client.Get(context.TODO(), name, partition)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		err = r.addPartition(protocol, cluster, partitionID)
		if err != nil {
			return err
		}
		err = r.client.Get(context.TODO(), name, partition)
		if err != nil {
			return err
		}
	}
	return r.reconcileMembers(protocol, cluster, partition)
}

func (r *Reconciler) addPartition(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster, partitionID int) error {
	partition := &storagev2beta1.RaftPartition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: protocol.Namespace,
			Name:      getPartitionName(protocol, int(cluster.Spec.ClusterID), partitionID),
		},
		Spec: storagev2beta1.RaftPartitionSpec{
			ClusterID:   cluster.Spec.ClusterID,
			PartitionID: int32(partitionID),
		},
	}
	if err := controllerutil.SetControllerReference(cluster, partition, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), partition)
}

func (r *Reconciler) reconcileMembers(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster, partition *storagev2beta1.RaftPartition) error {
	for memberID := range getMembers(protocol, int(cluster.Spec.ClusterID), int(partition.Spec.PartitionID)) {
		err := r.reconcileMember(protocol, cluster, partition, memberID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) reconcileMember(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster, partition *storagev2beta1.RaftPartition, memberID int) error {
	log.Info("Reconcile raft protocol member")
	member := &storagev2beta1.RaftMember{}
	name := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getMemberName(protocol, int(cluster.Spec.ClusterID), int(partition.Spec.PartitionID), memberID),
	}
	err := r.client.Get(context.TODO(), name, member)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		return r.addMember(protocol, cluster, partition, memberID)
	}
	return nil
}

func (r *Reconciler) addMember(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster, partition *storagev2beta1.RaftPartition, memberID int) error {
	member := &storagev2beta1.RaftMember{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: protocol.Namespace,
			Name:      getMemberName(protocol, int(cluster.Spec.ClusterID), int(partition.Spec.PartitionID), memberID),
		},
		Spec: storagev2beta1.RaftMemberSpec{
			ClusterID:   cluster.Spec.ClusterID,
			PartitionID: partition.Spec.PartitionID,
			MemberID:    int32(memberID),
			Pod:         getPodName(protocol, int(cluster.Spec.ClusterID), memberID),
		},
	}
	if err := controllerutil.SetControllerReference(partition, member, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), member)
}

func (r *Reconciler) startMonitoringCluster(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) error {
	for replicaID := range getReplicas(protocol, int(cluster.Spec.ClusterID)) {
		err := r.startMonitoringPod(protocol, cluster, replicaID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) startMonitoringPod(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster, podID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	replicaID := getPodName(protocol, int(cluster.Spec.ClusterID), podID)
	_, ok := r.streams[replicaID]
	if ok {
		return nil
	}

	pod := &corev1.Pod{}
	podName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      replicaID,
	}
	if err := r.client.Get(context.TODO(), podName, pod); err != nil {
		return err
	}

	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", pod.Status.PodIP, monitoringPort),
		grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := storage.NewRaftEventsClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	stream, err := client.Subscribe(ctx, &storage.SubscribeRequest{})
	if err != nil {
		cancel()
		return err
	}

	r.streams[replicaID] = cancel
	go func() {
		defer func() {
			cancel()
			r.mu.Lock()
			delete(r.streams, replicaID)
			r.mu.Unlock()
			go r.startMonitoringPod(protocol, cluster, podID)
		}()

		for {
			event, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return
			}

			log.Infof("Received event %+v from %s", event, replicaID)
			switch e := event.Event.(type) {
			case *storage.RaftEvent_MemberReady:
				r.recordPartitionReady(protocol, int(cluster.Spec.ClusterID), podID, e.MemberReady, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_LeaderUpdated:
				r.recordLeaderUpdated(protocol, int(cluster.Spec.ClusterID), podID, e.LeaderUpdated, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_MembershipChanged:
				r.recordMembershipChanged(protocol, int(cluster.Spec.ClusterID), podID, e.MembershipChanged, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_SendSnapshotStarted:
				r.recordSendSnapshotStarted(protocol, int(cluster.Spec.ClusterID), podID, e.SendSnapshotStarted, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_SendSnapshotCompleted:
				r.recordSendSnapshotCompleted(protocol, int(cluster.Spec.ClusterID), podID, e.SendSnapshotCompleted, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_SendSnapshotAborted:
				r.recordSendSnapshotAborted(protocol, int(cluster.Spec.ClusterID), podID, e.SendSnapshotAborted, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_SnapshotReceived:
				r.recordSnapshotReceived(protocol, int(cluster.Spec.ClusterID), podID, e.SnapshotReceived, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_SnapshotRecovered:
				r.recordSnapshotRecovered(protocol, int(cluster.Spec.ClusterID), podID, e.SnapshotRecovered, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_SnapshotCreated:
				r.recordSnapshotCreated(protocol, int(cluster.Spec.ClusterID), podID, e.SnapshotCreated, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_SnapshotCompacted:
				r.recordSnapshotCompacted(protocol, int(cluster.Spec.ClusterID), podID, e.SnapshotCompacted, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_LogCompacted:
				r.recordLogCompacted(protocol, int(cluster.Spec.ClusterID), podID, e.LogCompacted, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_LogdbCompacted:
				r.recordLogDBCompacted(protocol, int(cluster.Spec.ClusterID), podID, e.LogdbCompacted, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_ConnectionEstablished:
				r.recordConnectionEstablished(protocol, int(cluster.Spec.ClusterID), podID, e.ConnectionEstablished, metav1.NewTime(event.Timestamp))
			case *storage.RaftEvent_ConnectionFailed:
				r.recordConnectionFailed(protocol, int(cluster.Spec.ClusterID), podID, e.ConnectionFailed, metav1.NewTime(event.Timestamp))
			}
		}
	}()
	return nil
}

func (r *Reconciler) reconcileConfigMap(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) error {
	log.Info("Reconcile raft protocol config map")
	cm := &corev1.ConfigMap{}
	name := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getClusterName(protocol, int(cluster.Spec.ClusterID)),
	}
	err := r.client.Get(context.TODO(), name, cm)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addConfigMap(protocol, cluster)
	}
	return err
}

func (r *Reconciler) addConfigMap(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) error {
	log.Info("Creating raft ConfigMap", "Name", protocol.Name, "Namespace", protocol.Namespace)
	var config interface{}

	clusterConfig, err := newNodeConfigString(protocol, cluster)
	if err != nil {
		return err
	}

	protocolConfig, err := newProtocolConfigString(config)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getClusterName(protocol, int(cluster.Spec.ClusterID)),
			Namespace: protocol.Namespace,
			Labels:    newClusterLabels(protocol, int(cluster.Spec.ClusterID)),
		},
		Data: map[string]string{
			clusterConfigFile:  clusterConfig,
			protocolConfigFile: protocolConfig,
		},
	}

	if err := controllerutil.SetControllerReference(protocol, cm, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), cm)
}

// newNodeConfigString creates a node configuration string for the given cluster
func newNodeConfigString(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) (string, error) {
	replicaIDs := getReplicas(protocol, int(cluster.Spec.ClusterID))
	replicas := make([]protocolapi.ProtocolReplica, len(replicaIDs))
	for i, replicaID := range replicaIDs {
		replicas[i] = protocolapi.ProtocolReplica{
			ID:      replicaID,
			Host:    getPodDNSName(protocol, int(cluster.Spec.ClusterID), i),
			APIPort: apiPort,
			ExtraPorts: map[string]int32{
				protocolPortName: protocolPort,
			},
		}
	}

	partitionIDs := getPartitions(protocol, int(cluster.Spec.ClusterID))
	partitions := make([]protocolapi.ProtocolPartition, len(partitionIDs))
	for i, partitionID := range partitionIDs {
		partitions[i] = protocolapi.ProtocolPartition{
			PartitionID: uint32(partitionID),
			Replicas:    replicaIDs,
		}
	}

	config := &protocolapi.ProtocolConfig{
		Replicas:   replicas,
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

func (r *Reconciler) reconcileStatefulSet(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) error {
	log.Info("Reconcile raft protocol stateful set")
	statefulSet := &appsv1.StatefulSet{}
	name := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getClusterName(protocol, int(cluster.Spec.ClusterID)),
	}
	err := r.client.Get(context.TODO(), name, statefulSet)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addStatefulSet(protocol, cluster)
	}
	return err
}

func (r *Reconciler) addStatefulSet(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) error {
	log.Info("Creating raft replicas", "Name", protocol.Name, "Namespace", protocol.Namespace)

	image := getImage(protocol)
	pullPolicy := protocol.Spec.ImagePullPolicy
	if pullPolicy == "" {
		pullPolicy = corev1.PullIfNotPresent
	}

	volumes := []corev1.Volume{
		{
			Name: configVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: getClusterName(protocol, int(cluster.Spec.ClusterID)),
					},
				},
			},
		},
	}

	var volumeClaimTemplates []corev1.PersistentVolumeClaim

	dataVolumeName := dataVolume
	if protocol.Spec.VolumeClaimTemplate != nil {
		pvc := protocol.Spec.VolumeClaimTemplate
		if pvc.Name == "" {
			pvc.Name = dataVolume
		} else {
			dataVolumeName = pvc.Name
		}
		volumeClaimTemplates = append(volumeClaimTemplates, *pvc)
	} else {
		volumes = append(volumes, corev1.Volume{
			Name: dataVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	set := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getClusterName(protocol, int(cluster.Spec.ClusterID)),
			Namespace: protocol.Namespace,
			Labels:    newClusterLabels(protocol, int(cluster.Spec.ClusterID)),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: getClusterHeadlessServiceName(protocol, int(cluster.Spec.ClusterID)),
			Replicas:    &protocol.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: newClusterLabels(protocol, int(cluster.Spec.ClusterID)),
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: newClusterLabels(protocol, int(cluster.Spec.ClusterID)),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "raft",
							Image:           image,
							ImagePullPolicy: pullPolicy,
							Env: []corev1.EnvVar{
								{
									Name: "NODE_ID",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "api",
									ContainerPort: apiPort,
								},
								{
									Name:          "protocol",
									ContainerPort: protocolPort,
								},
							},
							Args: []string{
								"$(NODE_ID)",
								fmt.Sprintf("%s/%s", configPath, clusterConfigFile),
								fmt.Sprintf("%s/%s", configPath, protocolConfigFile),
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"stat", "/tmp/atomix-ready"},
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      10,
								FailureThreshold:    12,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{Type: intstr.Int, IntVal: probePort},
									},
								},
								InitialDelaySeconds: 60,
								TimeoutSeconds:      10,
							},
							SecurityContext: protocol.Spec.SecurityContext,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dataVolumeName,
									MountPath: dataPath,
								},
								{
									Name:      configVolume,
									MountPath: configPath,
								},
							},
						},
					},
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 1,
									PodAffinityTerm: corev1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: newClusterLabels(protocol, int(cluster.Spec.ClusterID)),
										},
										Namespaces:  []string{protocol.Namespace},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					ImagePullSecrets: protocol.Spec.ImagePullSecrets,
					Volumes:          volumes,
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}

	if err := controllerutil.SetControllerReference(protocol, set, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), set)
}

func (r *Reconciler) reconcileService(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) error {
	log.Info("Reconcile raft protocol headless service")
	service := &corev1.Service{}
	name := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getClusterHeadlessServiceName(protocol, int(cluster.Spec.ClusterID)),
	}
	err := r.client.Get(context.TODO(), name, service)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addService(protocol, cluster)
	}
	return err
}

func (r *Reconciler) addService(protocol *storagev2beta1.MultiRaftProtocol, cluster *storagev2beta1.RaftCluster) error {
	log.Info("Creating headless raft service", "Name", protocol.Name, "Namespace", protocol.Namespace)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getClusterHeadlessServiceName(protocol, int(cluster.Spec.ClusterID)),
			Namespace: protocol.Namespace,
			Labels:    newClusterLabels(protocol, int(cluster.Spec.ClusterID)),
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
			Selector:                 newClusterLabels(protocol, int(cluster.Spec.ClusterID)),
		},
	}

	if err := controllerutil.SetControllerReference(protocol, service, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), service)
}

func (r *Reconciler) getCluster(protocol *storagev2beta1.MultiRaftProtocol, clusterID int) (*storagev2beta1.RaftCluster, error) {
	clusterName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getClusterName(protocol, clusterID),
	}
	cluster := &storagev2beta1.RaftCluster{}
	if err := r.client.Get(context.TODO(), clusterName, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func (r *Reconciler) getPartition(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, partitionID int) (*storagev2beta1.RaftPartition, error) {
	partitionName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getPartitionName(protocol, clusterID, partitionID),
	}
	partition := &storagev2beta1.RaftPartition{}
	if err := r.client.Get(context.TODO(), partitionName, partition); err != nil {
		return nil, err
	}
	return partition, nil
}

func (r *Reconciler) getMember(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, partitionID int, memberID int) (*storagev2beta1.RaftMember, error) {
	memberName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getMemberName(protocol, clusterID, partitionID, memberID),
	}
	member := &storagev2beta1.RaftMember{}
	if err := r.client.Get(context.TODO(), memberName, member); err != nil {
		return nil, err
	}
	return member, nil
}

func (r *Reconciler) getPod(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int) (*corev1.Pod, error) {
	podName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getPodName(protocol, clusterID, podID),
	}
	pod := &corev1.Pod{}
	if err := r.client.Get(context.TODO(), podName, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

// getNumClusters returns the number of clusters in the given database
func getNumClusters(protocol *storagev2beta1.MultiRaftProtocol) int {
	if protocol.Spec.Clusters == 0 {
		return 1
	}
	return int(protocol.Spec.Clusters)
}

func getClusters(protocol *storagev2beta1.MultiRaftProtocol) []int {
	numClusters := getNumClusters(protocol)
	clusters := make([]int, numClusters)
	for i := 0; i < numClusters; i++ {
		clusters[i] = i + 1
	}
	return clusters
}

// getNumPartitions returns the number of replicas in the given database
func getNumPartitions(protocol *storagev2beta1.MultiRaftProtocol) int {
	if protocol.Spec.Partitions == 0 {
		return 1
	}
	return int(protocol.Spec.Partitions)
}

func getPartitions(protocol *storagev2beta1.MultiRaftProtocol, clusterID int) []int {
	numPartitions := getNumPartitions(protocol)
	partitions := make([]int, 0, numPartitions)
	for partitionID := 1; partitionID <= numPartitions; partitionID++ {
		if getClusterForPartitionID(protocol, partitionID) == clusterID {
			partitions = append(partitions, partitionID)
		}
	}
	return partitions
}

// getNumReplicas returns the number of replicas in the given database
func getNumReplicas(protocol *storagev2beta1.MultiRaftProtocol) int {
	if protocol.Spec.Replicas == 0 {
		return 1
	}
	return int(protocol.Spec.Replicas)
}

func getReplicas(protocol *storagev2beta1.MultiRaftProtocol, clusterID int) []string {
	numReplicas := getNumReplicas(protocol)
	replicas := make([]string, numReplicas)
	for i := 0; i < numReplicas; i++ {
		replicas[i] = getPodName(protocol, clusterID, i)
	}
	return replicas
}

func getMembers(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, partitionID int) []string {
	numMembers := getNumReplicas(protocol)
	members := make([]string, numMembers)
	for i := 0; i < numMembers; i++ {
		members[i] = getMemberName(protocol, clusterID, partitionID, i)
	}
	return members
}

// getClusterForPartitionID returns the cluster ID for the given partition ID
func getClusterForPartitionID(protocol *storagev2beta1.MultiRaftProtocol, partitionID int) int {
	return (partitionID % getNumClusters(protocol)) + 1
}

// getClusterResourceName returns the given resource name for the given cluster
func getClusterResourceName(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, resource string) string {
	return fmt.Sprintf("%s-%s", getClusterName(protocol, clusterID), resource)
}

// getClusterName returns the cluster name
func getClusterName(protocol *storagev2beta1.MultiRaftProtocol, clusterID int) string {
	return fmt.Sprintf("%s-%d", protocol.Name, clusterID)
}

// getPartitionName returns the partition name
func getPartitionName(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, partitionID int) string {
	return fmt.Sprintf("%s-%d", getClusterName(protocol, clusterID), partitionID)
}

// getMemberName returns the member name
func getMemberName(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, partitionID int, podID int) string {
	return fmt.Sprintf("%s-%d", getPartitionName(protocol, clusterID, partitionID), podID)
}

// getClusterHeadlessServiceName returns the headless service name for the given cluster
func getClusterHeadlessServiceName(protocol *storagev2beta1.MultiRaftProtocol, clusterID int) string {
	return getClusterResourceName(protocol, clusterID, headlessServiceSuffix)
}

// getPodName returns the name of the pod for the given pod ID
func getPodName(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int) string {
	return fmt.Sprintf("%s-%d", getClusterName(protocol, clusterID), podID)
}

// getPodDNSName returns the fully qualified DNS name for the given pod ID
func getPodDNSName(protocol *storagev2beta1.MultiRaftProtocol, clusterID int, podID int) string {
	domain := os.Getenv(clusterDomainEnv)
	if domain == "" {
		domain = "cluster.local"
	}
	return fmt.Sprintf("%s-%d.%s.%s.svc.%s", getClusterName(protocol, clusterID), podID, getClusterHeadlessServiceName(protocol, clusterID), protocol.Namespace, domain)
}

// newClusterLabels returns the labels for the given cluster
func newClusterLabels(protocol *storagev2beta1.MultiRaftProtocol, clusterID int) map[string]string {
	labels := make(map[string]string)
	for key, value := range protocol.Labels {
		labels[key] = value
	}
	labels[appLabel] = appAtomix
	labels[databaseLabel] = fmt.Sprintf("%s.%s", protocol.Name, protocol.Namespace)
	labels[clusterLabel] = fmt.Sprint(clusterID)
	return labels
}

func getImage(protocol *storagev2beta1.MultiRaftProtocol) string {
	if protocol.Spec.Image != "" {
		return protocol.Spec.Image
	}
	return getDefaultImage()
}

func getDefaultImage() string {
	image := os.Getenv(defaultImageEnv)
	if image == "" {
		image = defaultImage
	}
	return image
}
