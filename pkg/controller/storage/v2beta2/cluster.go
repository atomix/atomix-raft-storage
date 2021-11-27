// Copyright 2021-present Open Networking Foundation.
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

package v2beta2

import (
	"context"
	"encoding/json"
	"fmt"
	protocolapi "github.com/atomix/atomix-api/go/atomix/protocol"
	corev2beta1 "github.com/atomix/atomix-controller/pkg/apis/core/v2beta1"
	raftconfig "github.com/atomix/atomix-raft-storage/pkg/storage/config"
	"github.com/gogo/protobuf/jsonpb"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"
	"net"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"

	storagev2beta2 "github.com/atomix/atomix-raft-storage/pkg/apis/storage/v2beta2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

func addMultiRaftClusterController(mgr manager.Manager) error {
	options := controller.Options{
		Reconciler: &MultiRaftClusterReconciler{
			client: mgr.GetClient(),
			scheme: mgr.GetScheme(),
			events: mgr.GetEventRecorderFor("atomix-raft-storage"),
		},
		RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Millisecond*10, time.Second*5),
	}

	// Create a new controller
	controller, err := controller.New("atomix-raft-cluster-v2beta2", mgr, options)
	if err != nil {
		return err
	}

	// Watch for changes to the storage resource and enqueue Clusters that reference it
	err = controller.Watch(&source.Kind{Type: &storagev2beta2.MultiRaftCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource StatefulSet
	err = controller.Watch(&source.Kind{Type: &storagev2beta2.RaftGroup{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &storagev2beta2.MultiRaftCluster{},
		IsController: true,
	})
	if err != nil {
		return err
	}
	return nil
}

// MultiRaftClusterReconciler reconciles a MultiRaftCluster object
type MultiRaftClusterReconciler struct {
	client client.Client
	scheme *runtime.Scheme
	events record.EventRecorder
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
func (r *MultiRaftClusterReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconcile MultiRaftCluster")
	cluster := &storagev2beta2.MultiRaftCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cluster)
	if err != nil {
		log.Error(err, "Reconcile MultiRaftCluster")
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	ok, err := r.reconcileGroups(cluster)
	if err != nil {
		log.Error(err, "Reconcile Cluster")
		return reconcile.Result{}, err
	} else if ok {
		return reconcile.Result{}, nil
	}

	err = r.reconcileConfigMap(cluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.reconcileStatefulSet(cluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.reconcileService(cluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.reconcileHeadlessService(cluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	ok, err = r.reconcileStatus(cluster)
	if err != nil {
		log.Error(err, "Reconcile Cluster")
		return reconcile.Result{}, err
	} else if ok {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}

func (r *MultiRaftClusterReconciler) reconcileGroups(cluster *storagev2beta2.MultiRaftCluster) (bool, error) {
	for i := 1; i <= getNumPartitions(cluster); i++ {
		if ok, err := r.reconcileGroup(cluster, i); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
	}
	return false, nil
}

func (r *MultiRaftClusterReconciler) reconcileGroup(cluster *storagev2beta2.MultiRaftCluster, groupID int) (bool, error) {
	group := &storagev2beta2.RaftGroup{}
	groupName := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      fmt.Sprintf("%s-%d", cluster.Name, groupID),
	}
	if err := r.client.Get(context.TODO(), groupName, group); err != nil {
		if !k8serrors.IsNotFound(err) {
			return false, err
		}
		group = &storagev2beta2.RaftGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: groupName.Namespace,
				Name:      groupName.Name,
				Labels:    cluster.Labels,
			},
			Spec: storagev2beta2.RaftGroupSpec{
				RaftConfig: cluster.Spec.Raft,
				GroupID:    int32(groupID),
			},
		}
		if err := controllerutil.SetControllerReference(cluster, group, r.scheme); err != nil {
			return false, err
		}
		if err := r.client.Create(context.TODO(), group); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *MultiRaftClusterReconciler) reconcileConfigMap(cluster *storagev2beta2.MultiRaftCluster) error {
	log.Info("Reconcile raft protocol config map")
	cm := &corev1.ConfigMap{}
	name := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}
	err := r.client.Get(context.TODO(), name, cm)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addConfigMap(cluster)
	}
	return err
}

func (r *MultiRaftClusterReconciler) addConfigMap(cluster *storagev2beta2.MultiRaftCluster) error {
	log.Info("Creating raft ConfigMap", "Name", cluster.Name, "Namespace", cluster.Namespace)

	clusterConfig, err := newNodeConfigString(cluster)
	if err != nil {
		return err
	}

	protocolConfig, err := newProtocolConfigString(cluster)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    newClusterLabels(cluster),
		},
		Data: map[string]string{
			clusterConfigFile:  clusterConfig,
			protocolConfigFile: protocolConfig,
		},
	}

	if err := controllerutil.SetControllerReference(cluster, cm, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), cm)
}

// newNodeConfigString creates a node configuration string for the given cluster
func newNodeConfigString(cluster *storagev2beta2.MultiRaftCluster) (string, error) {
	numReplicas := getNumReplicas(cluster)
	replicas := make([]protocolapi.ProtocolReplica, numReplicas)
	for i := 0; i < numReplicas; i++ {
		replicas[i] = protocolapi.ProtocolReplica{
			ID:      fmt.Sprintf("%s-%d", cluster.Name, i),
			Host:    getPodDNSName(cluster, i),
			APIPort: apiPort,
			ExtraPorts: map[string]int32{
				protocolPortName: protocolPort,
			},
		}
	}

	numPartitions := getNumPartitions(cluster)
	partitions := make([]protocolapi.ProtocolPartition, numPartitions)
	for i := 0; i < numPartitions; i++ {
		partitionID := i + 1
		numMembers := getNumMembers(cluster)
		numROMembers := getNumROMembers(cluster)
		members := make([]string, numMembers)
		for j := 0; j < numMembers; j++ {
			members[j] = fmt.Sprintf("%s-%d", cluster.Name, (((numMembers+numROMembers)*partitionID)+j)%numReplicas)
		}
		roMembers := make([]string, numROMembers)
		for j := 0; j < numROMembers; j++ {
			members[j] = fmt.Sprintf("%s-%d", cluster.Name, (((numMembers+numROMembers)*partitionID)+numMembers+j)%numReplicas)
		}
		partitions[i] = protocolapi.ProtocolPartition{
			PartitionID:  uint32(partitionID),
			Replicas:     members,
			ReadReplicas: roMembers,
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
func newProtocolConfigString(cluster *storagev2beta2.MultiRaftCluster) (string, error) {
	config := raftconfig.ProtocolConfig{}

	electionTimeout := cluster.Spec.Raft.ElectionTimeout
	if electionTimeout != nil {
		config.ElectionTimeout = &electionTimeout.Duration
	}

	heartbeatPeriod := cluster.Spec.Raft.HeartbeatPeriod
	if heartbeatPeriod != nil {
		config.HeartbeatPeriod = &heartbeatPeriod.Duration
	}

	entryThreshold := cluster.Spec.Raft.SnapshotEntryThreshold
	if entryThreshold != nil {
		config.SnapshotEntryThreshold = uint64(*entryThreshold)
	} else {
		config.SnapshotEntryThreshold = 10000
	}

	retainEntries := cluster.Spec.Raft.CompactionRetainEntries
	if retainEntries != nil {
		config.CompactionRetainEntries = uint64(*retainEntries)
	} else {
		config.CompactionRetainEntries = 1000
	}

	bytes, err := json.Marshal(&config)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (r *MultiRaftClusterReconciler) reconcileStatefulSet(cluster *storagev2beta2.MultiRaftCluster) error {
	log.Info("Reconcile raft protocol stateful set")
	statefulSet := &appsv1.StatefulSet{}
	name := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}
	err := r.client.Get(context.TODO(), name, statefulSet)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addStatefulSet(cluster)
	}
	return err
}

func (r *MultiRaftClusterReconciler) addStatefulSet(cluster *storagev2beta2.MultiRaftCluster) error {
	log.Info("Creating raft replicas", "Name", cluster.Name, "Namespace", cluster.Namespace)

	image := getImage(cluster)
	pullPolicy := cluster.Spec.ImagePullPolicy
	if pullPolicy == "" {
		pullPolicy = corev1.PullIfNotPresent
	}

	volumes := []corev1.Volume{
		{
			Name: configVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cluster.Name,
					},
				},
			},
		},
	}

	var volumeClaimTemplates []corev1.PersistentVolumeClaim

	dataVolumeName := dataVolume
	if cluster.Spec.VolumeClaimTemplate != nil {
		pvc := cluster.Spec.VolumeClaimTemplate
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
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    newClusterLabels(cluster),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: getClusterHeadlessServiceName(cluster),
			Replicas:    pointer.Int32Ptr(int32(getNumReplicas(cluster))),
			Selector: &metav1.LabelSelector{
				MatchLabels: newClusterLabels(cluster),
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: newClusterLabels(cluster),
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
							SecurityContext: cluster.Spec.SecurityContext,
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
											MatchLabels: newClusterLabels(cluster),
										},
										Namespaces:  []string{cluster.Namespace},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					ImagePullSecrets: cluster.Spec.ImagePullSecrets,
					Volumes:          volumes,
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}

	if err := controllerutil.SetControllerReference(cluster, set, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), set)
}

func (r *MultiRaftClusterReconciler) reconcileService(cluster *storagev2beta2.MultiRaftCluster) error {
	log.Info("Reconcile raft protocol service")
	service := &corev1.Service{}
	name := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}
	err := r.client.Get(context.TODO(), name, service)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addService(cluster)
	}
	return err
}

func (r *MultiRaftClusterReconciler) addService(cluster *storagev2beta2.MultiRaftCluster) error {
	log.Info("Creating raft service", "Name", cluster.Name, "Namespace", cluster.Namespace)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    newClusterLabels(cluster),
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
			Selector: newClusterLabels(cluster),
		},
	}

	if err := controllerutil.SetControllerReference(cluster, service, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), service)
}

func (r *MultiRaftClusterReconciler) reconcileHeadlessService(cluster *storagev2beta2.MultiRaftCluster) error {
	log.Info("Reconcile raft protocol headless service")
	service := &corev1.Service{}
	name := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      getClusterHeadlessServiceName(cluster),
	}
	err := r.client.Get(context.TODO(), name, service)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addHeadlessService(cluster)
	}
	return err
}

func (r *MultiRaftClusterReconciler) addHeadlessService(cluster *storagev2beta2.MultiRaftCluster) error {
	log.Info("Creating headless raft service", "Name", cluster.Name, "Namespace", cluster.Namespace)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getClusterHeadlessServiceName(cluster),
			Namespace: cluster.Namespace,
			Labels:    newClusterLabels(cluster),
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
			Selector:                 newClusterLabels(cluster),
		},
	}

	if err := controllerutil.SetControllerReference(cluster, service, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), service)
}

func (r *MultiRaftClusterReconciler) reconcileStatus(cluster *storagev2beta2.MultiRaftCluster) (bool, error) {
	replicas, err := r.getProtocolReplicas(cluster)
	if err != nil {
		return false, err
	}

	partitions, err := r.getProtocolPartitions(cluster)
	if err != nil {
		return false, err
	}

	state := storagev2beta2.MultiRaftClusterReady
	for _, replica := range replicas {
		if !replica.Ready {
			state = storagev2beta2.MultiRaftClusterNotReady
			break
		}
	}
	for _, partition := range partitions {
		if !partition.Ready {
			state = storagev2beta2.MultiRaftClusterNotReady
			break
		}
	}

	if cluster.Status.State != state {
		cluster.Status.State = state
		if err := r.client.Status().Update(context.TODO(), cluster); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *MultiRaftClusterReconciler) getProtocolReplicas(cluster *storagev2beta2.MultiRaftCluster) ([]corev2beta1.ReplicaStatus, error) {
	numReplicas := getNumReplicas(cluster)
	replicas := make([]corev2beta1.ReplicaStatus, numReplicas)
	for i := 0; i < numReplicas; i++ {
		replicaReady, err := r.isReplicaReady(cluster, i)
		if err != nil {
			return nil, err
		}
		replica := corev2beta1.ReplicaStatus{
			ID:   fmt.Sprintf("%s-%d", cluster.Name, i),
			Host: pointer.StringPtr(getPodDNSName(cluster, i)),
			Port: pointer.Int32Ptr(int32(apiPort)),
			ExtraPorts: map[string]int32{
				protocolPortName: protocolPort,
			},
			Ready: replicaReady,
		}
		replicas[i] = replica
	}
	return replicas, nil
}

func (r *MultiRaftClusterReconciler) getProtocolPartitions(cluster *storagev2beta2.MultiRaftCluster) ([]corev2beta1.PartitionStatus, error) {
	numReplicas := getNumReplicas(cluster)
	numPartitions := getNumPartitions(cluster)
	partitions := make([]corev2beta1.PartitionStatus, numPartitions)
	for i := 0; i < numPartitions; i++ {
		partitionID := i + 1
		partitionReady := true
		numMembers := getNumMembers(cluster)
		numROMembers := getNumROMembers(cluster)

		replicas := make([]string, numMembers)
		for j := 0; j < numMembers; j++ {
			replicaID := (((numMembers + numROMembers) * partitionID) + j) % numReplicas
			replicaReady, err := r.isReplicaReady(cluster, replicaID)
			if err != nil {
				return nil, err
			} else if !replicaReady {
				partitionReady = false
			}
			replicas[j] = fmt.Sprintf("%s-%d", cluster.Name, replicaID)
		}

		roReplicas := make([]string, numROMembers)
		for j := 0; j < numROMembers; j++ {
			replicaID := (((numMembers + numROMembers) * partitionID) + numMembers + j) % numReplicas
			replicaReady, err := r.isReplicaReady(cluster, replicaID)
			if err != nil {
				return nil, err
			} else if !replicaReady {
				partitionReady = false
			}
			roReplicas[j] = fmt.Sprintf("%s-%d", cluster.Name, replicaID)
		}
		partition := corev2beta1.PartitionStatus{
			ID:           uint32(partitionID),
			Host:         pointer.StringPtr(fmt.Sprintf("%s.%s", cluster.Name, cluster.Namespace)),
			Port:         pointer.Int32Ptr(apiPort),
			Replicas:     replicas,
			ReadReplicas: roReplicas,
			Ready:        partitionReady,
		}
		partitions[i] = partition
	}
	return partitions, nil
}

func (r *MultiRaftClusterReconciler) isReplicaReady(cluster *storagev2beta2.MultiRaftCluster, replica int) (bool, error) {
	podName := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      fmt.Sprintf("%s-%d", cluster.Name, replica),
	}
	pod := &corev1.Pod{}
	if err := r.client.Get(context.TODO(), podName, pod); err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue, nil
		}
	}
	return false, nil
}

var _ reconcile.Reconciler = (*MultiRaftClusterReconciler)(nil)

func getNumPartitions(cluster *storagev2beta2.MultiRaftCluster) int {
	if cluster.Spec.Groups == 0 {
		return 1
	}
	return int(cluster.Spec.Groups)
}

func getNumReplicas(cluster *storagev2beta2.MultiRaftCluster) int {
	if cluster.Spec.Replicas == 0 {
		return 1
	}
	return int(cluster.Spec.Replicas)
}

func getNumMembers(cluster *storagev2beta2.MultiRaftCluster) int {
	if cluster.Spec.Raft.QuorumSize == nil {
		return getNumReplicas(cluster)
	}
	return int(*cluster.Spec.Raft.QuorumSize)
}

func getNumROMembers(cluster *storagev2beta2.MultiRaftCluster) int {
	if cluster.Spec.Raft.ReadReplicas == nil {
		return 0
	}
	return int(*cluster.Spec.Raft.ReadReplicas)
}

// getClusterResourceName returns the given resource name for the given cluster
func getClusterResourceName(cluster *storagev2beta2.MultiRaftCluster, resource string) string {
	return fmt.Sprintf("%s-%s", cluster.Name, resource)
}

// getClusterHeadlessServiceName returns the headless service name for the given cluster
func getClusterHeadlessServiceName(cluster *storagev2beta2.MultiRaftCluster) string {
	return getClusterResourceName(cluster, headlessServiceSuffix)
}

// GetClusterDomain returns Kubernetes cluster domain, default to "cluster.local"
func getClusterDomain() string {
	apiSvc := "kubernetes.default.svc"
	cname, err := net.LookupCNAME(apiSvc)
	if err != nil {
		defaultClusterDomain := "cluster.local"
		return defaultClusterDomain
	}
	return strings.TrimPrefix(cname, apiSvc + ".")
}

// getPodDNSName returns the fully qualified DNS name for the given pod ID
func getPodDNSName(cluster *storagev2beta2.MultiRaftCluster, podID int) string {
	return fmt.Sprintf("%s-%d.%s.%s.svc.%s", cluster.Name, podID, getClusterHeadlessServiceName(cluster), cluster.Namespace, getClusterDomain())
}

// newClusterLabels returns the labels for the given cluster
func newClusterLabels(cluster *storagev2beta2.MultiRaftCluster) map[string]string {
	labels := make(map[string]string)
	for key, value := range cluster.Labels {
		labels[key] = value
	}
	labels[appLabel] = appAtomix
	labels[clusterLabel] = cluster.Name
	return labels
}

func getImage(cluster *storagev2beta2.MultiRaftCluster) string {
	if cluster.Spec.Image != "" {
		return cluster.Spec.Image
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
