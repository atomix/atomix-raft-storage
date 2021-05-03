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
	corev2beta1 "github.com/atomix/atomix-controller/pkg/apis/core/v2beta1"
	"k8s.io/utils/pointer"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	storagev2beta1 "github.com/atomix/atomix-raft-storage-plugin/pkg/apis/storage/v2beta1"
	"github.com/gogo/protobuf/jsonpb"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	apiPort               = 5678
	protocolPortName      = "raft"
	protocolPort          = 5679
	probePort             = 5679
	defaultImageEnv       = "DEFAULT_NODE_IMAGE"
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

const clusterDomainEnv = "CLUSTER_DOMAIN"

func addRaftProtocolController(mgr manager.Manager) error {
	r := &Reconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}

	// Create a new controller
	c, err := controller.New(mgr.GetScheme().Name(), mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to the storage resource and enqueue Clusters that reference it
	err = c.Watch(&source.Kind{Type: &storagev2beta1.MultiRaftProtocol{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource StatefulSet
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &storagev2beta1.MultiRaftProtocol{},
		IsController: true,
	})
	if err != nil {
		return err
	}
	return nil
}

var _ reconcile.Reconciler = &Reconciler{}

// Reconciler reconciles a MultiRaftProtocol object
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconcile MultiRaftProtocol")
	protocol := &storagev2beta1.MultiRaftProtocol{}
	err := r.client.Get(context.TODO(), request.NamespacedName, protocol)
	if err != nil {
		log.Error(err, "Reconcile MultiRaftProtocol")
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	log.Info("Reconcile Clusters")
	err = r.reconcileClusters(protocol)
	if err != nil {
		log.Error(err, "Reconcile Clusters")
		return reconcile.Result{}, err
	}

	log.Info("Reconcile Protocol")
	err = r.reconcileStatus(protocol)
	if err != nil {
		log.Error(err, "Reconcile Protocol")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileStatus(protocol *storagev2beta1.MultiRaftProtocol) error {
	replicas, err := r.getProtocolReplicas(protocol)
	if err != nil {
		return err
	}

	partitions, err := r.getProtocolPartitions(protocol)
	if err != nil {
		return err
	}

	if protocol.Status.ProtocolStatus == nil ||
		isReplicasChanged(protocol.Status.Replicas, replicas) ||
		isPartitionsChanged(protocol.Status.Partitions, partitions) {
		var revision int64
		if protocol.Status.ProtocolStatus != nil {
			revision = protocol.Status.ProtocolStatus.Revision
		}
		if protocol.Status.ProtocolStatus == nil ||
			!isReplicasSame(protocol.Status.Replicas, replicas) ||
			!isPartitionsSame(protocol.Status.Partitions, partitions) {
			revision++
		}
		protocol.Status.ProtocolStatus = &corev2beta1.ProtocolStatus{
			Revision:   revision,
			Replicas:   replicas,
			Partitions: partitions,
		}
		return r.client.Status().Update(context.TODO(), protocol)
	}
	return nil
}

func isReplicasSame(a, b []corev2beta1.ReplicaStatus) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ar := a[i]
		br := b[i]
		if ar.ID != br.ID {
			return false
		}
	}
	return true
}

func isReplicasChanged(a, b []corev2beta1.ReplicaStatus) bool {
	if len(a) != len(b) {
		return true
	}
	for i := 0; i < len(a); i++ {
		ar := a[i]
		br := b[i]
		if ar.ID != br.ID {
			return true
		}
		if ar.Ready != br.Ready {
			return true
		}
	}
	return false
}

func isPartitionsSame(a, b []corev2beta1.PartitionStatus) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ap := a[i]
		bp := b[i]
		if ap.ID != bp.ID {
			return false
		}
		if len(ap.Replicas) != len(bp.Replicas) {
			return false
		}
		for j := 0; j < len(ap.Replicas); j++ {
			if ap.Replicas[j] != bp.Replicas[j] {
				return false
			}
		}
	}
	return true
}

func isPartitionsChanged(a, b []corev2beta1.PartitionStatus) bool {
	if len(a) != len(b) {
		return true
	}
	for i := 0; i < len(a); i++ {
		ap := a[i]
		bp := b[i]
		if ap.ID != bp.ID {
			return true
		}
		if len(ap.Replicas) != len(bp.Replicas) {
			return true
		}
		for j := 0; j < len(ap.Replicas); j++ {
			if ap.Replicas[j] != bp.Replicas[j] {
				return true
			}
		}
		if ap.Ready != bp.Ready {
			return true
		}
	}
	return false
}

func (r *Reconciler) getProtocolReplicas(protocol *storagev2beta1.MultiRaftProtocol) ([]corev2beta1.ReplicaStatus, error) {
	numClusters := getClusters(protocol)
	numReplicas := getReplicas(protocol)
	replicas := make([]corev2beta1.ReplicaStatus, 0, numReplicas*numClusters)
	for i := 1; i <= numClusters; i++ {
		for j := 0; j < numReplicas; j++ {
			replicaReady, err := r.isReplicaReady(protocol, i, j)
			if err != nil {
				return nil, err
			}
			replica := corev2beta1.ReplicaStatus{
				ID:   getPodName(protocol, i, j),
				Host: pointer.StringPtr(getPodDNSName(protocol, i, j)),
				Port: pointer.Int32Ptr(int32(apiPort)),
				ExtraPorts: map[string]int32{
					protocolPortName: protocolPort,
				},
				Ready: replicaReady,
			}
			replicas = append(replicas, replica)
		}
	}
	return replicas, nil
}

func (r *Reconciler) getProtocolPartitions(protocol *storagev2beta1.MultiRaftProtocol) ([]corev2beta1.PartitionStatus, error) {
	numClusters := getClusters(protocol)
	numReplicas := getReplicas(protocol)
	partitions := make([]corev2beta1.PartitionStatus, 0, protocol.Spec.Partitions)
	for partitionID := 1; partitionID <= int(protocol.Spec.Partitions); partitionID++ {
		for i := 1; i <= numClusters; i++ {
			partitionReady := true
			replicaNames := make([]string, 0, numReplicas)
			for j := 0; j < numReplicas; j++ {
				replicaReady, err := r.isReplicaReady(protocol, i, j)
				if err != nil {
					return nil, err
				} else if !replicaReady {
					partitionReady = false
				}
				replicaNames = append(replicaNames, getPodName(protocol, i, j))
			}
			partition := corev2beta1.PartitionStatus{
				ID:       uint32(partitionID),
				Replicas: replicaNames,
				Ready:    partitionReady,
			}
			partitions = append(partitions, partition)
		}
	}
	return partitions, nil
}

func (r *Reconciler) isReplicaReady(protocol *storagev2beta1.MultiRaftProtocol, cluster int, replica int) (bool, error) {
	podName := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getPodName(protocol, cluster, replica),
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

func (r *Reconciler) reconcileClusters(protocol *storagev2beta1.MultiRaftProtocol) error {
	clusters := getClusters(protocol)
	for cluster := 1; cluster <= clusters; cluster++ {
		err := r.reconcileCluster(protocol, cluster)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) reconcileCluster(protocol *storagev2beta1.MultiRaftProtocol, cluster int) error {
	err := r.reconcileConfigMap(protocol, cluster)
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
	return nil
}

func (r *Reconciler) reconcileConfigMap(protocol *storagev2beta1.MultiRaftProtocol, cluster int) error {
	log.Info("Reconcile raft protocol config map")
	cm := &corev1.ConfigMap{}
	name := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getClusterName(protocol, cluster),
	}
	err := r.client.Get(context.TODO(), name, cm)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addConfigMap(protocol, cluster)
	}
	return err
}

func (r *Reconciler) addConfigMap(protocol *storagev2beta1.MultiRaftProtocol, cluster int) error {
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
			Name:      getClusterName(protocol, cluster),
			Namespace: protocol.Namespace,
			Labels:    newClusterLabels(protocol, cluster),
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
func newNodeConfigString(protocol *storagev2beta1.MultiRaftProtocol, cluster int) (string, error) {
	replicas := make([]protocolapi.ProtocolReplica, protocol.Spec.Replicas)
	replicaNames := make([]string, protocol.Spec.Replicas)
	for i := 0; i < getReplicas(protocol); i++ {
		replicas[i] = protocolapi.ProtocolReplica{
			ID:      getPodName(protocol, cluster, i),
			Host:    getPodDNSName(protocol, cluster, i),
			APIPort: apiPort,
			ExtraPorts: map[string]int32{
				protocolPortName: protocolPort,
			},
		}
		replicaNames[i] = getPodName(protocol, cluster, i)
	}

	partitions := make([]protocolapi.ProtocolPartition, 0, protocol.Spec.Partitions)
	for partitionID := 1; partitionID <= int(protocol.Spec.Partitions); partitionID++ {
		if getClusterForPartitionID(protocol, partitionID) == cluster {
			partition := protocolapi.ProtocolPartition{
				PartitionID: uint32(partitionID),
				Replicas:    replicaNames,
			}
			partitions = append(partitions, partition)
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

func (r *Reconciler) reconcileStatefulSet(protocol *storagev2beta1.MultiRaftProtocol, cluster int) error {
	log.Info("Reconcile raft protocol stateful set")
	statefulSet := &appsv1.StatefulSet{}
	name := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getClusterName(protocol, cluster),
	}
	err := r.client.Get(context.TODO(), name, statefulSet)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addStatefulSet(protocol, cluster)
	}
	return err
}

func (r *Reconciler) addStatefulSet(protocol *storagev2beta1.MultiRaftProtocol, cluster int) error {
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
						Name: getClusterName(protocol, cluster),
					},
				},
			},
		},
	}

	volumeClaimTemplates := []corev1.PersistentVolumeClaim{}

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
			Name:      getClusterName(protocol, cluster),
			Namespace: protocol.Namespace,
			Labels:    newClusterLabels(protocol, cluster),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: getClusterHeadlessServiceName(protocol, cluster),
			Replicas:    &protocol.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: newClusterLabels(protocol, cluster),
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: newClusterLabels(protocol, cluster),
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
											MatchLabels: newClusterLabels(protocol, cluster),
										},
										Namespaces:  []string{protocol.Namespace},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					Volumes: volumes,
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

func (r *Reconciler) reconcileService(protocol *storagev2beta1.MultiRaftProtocol, cluster int) error {
	log.Info("Reconcile raft protocol headless service")
	service := &corev1.Service{}
	name := types.NamespacedName{
		Namespace: protocol.Namespace,
		Name:      getClusterHeadlessServiceName(protocol, cluster),
	}
	err := r.client.Get(context.TODO(), name, service)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addService(protocol, cluster)
	}
	return err
}

func (r *Reconciler) addService(protocol *storagev2beta1.MultiRaftProtocol, cluster int) error {
	log.Info("Creating headless raft service", "Name", protocol.Name, "Namespace", protocol.Namespace)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getClusterHeadlessServiceName(protocol, cluster),
			Namespace: protocol.Namespace,
			Labels:    newClusterLabels(protocol, cluster),
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
			Selector:                 newClusterLabels(protocol, cluster),
		},
	}

	if err := controllerutil.SetControllerReference(protocol, service, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), service)
}

// getClusters returns the number of clusters in the given database
func getClusters(storage *storagev2beta1.MultiRaftProtocol) int {
	if storage.Spec.Clusters == 0 {
		return 1
	}
	return int(storage.Spec.Clusters)
}

// getReplicas returns the number of replicas in the given database
func getReplicas(storage *storagev2beta1.MultiRaftProtocol) int {
	if storage.Spec.Replicas == 0 {
		return 1
	}
	return int(storage.Spec.Replicas)
}

// getClusterForPartitionID returns the cluster ID for the given partition ID
func getClusterForPartitionID(protocol *storagev2beta1.MultiRaftProtocol, partition int) int {
	return (partition % getClusters(protocol)) + 1
}

// getClusterResourceName returns the given resource name for the given cluster
func getClusterResourceName(protocol *storagev2beta1.MultiRaftProtocol, cluster int, resource string) string {
	return fmt.Sprintf("%s-%s", getClusterName(protocol, cluster), resource)
}

// getClusterName returns the cluster name
func getClusterName(protocol *storagev2beta1.MultiRaftProtocol, cluster int) string {
	return fmt.Sprintf("%s-%d", protocol.Name, cluster)
}

// getClusterHeadlessServiceName returns the headless service name for the given cluster
func getClusterHeadlessServiceName(protocol *storagev2beta1.MultiRaftProtocol, cluster int) string {
	return getClusterResourceName(protocol, cluster, headlessServiceSuffix)
}

// getPodName returns the name of the pod for the given pod ID
func getPodName(protocol *storagev2beta1.MultiRaftProtocol, cluster int, pod int) string {
	return fmt.Sprintf("%s-%d", getClusterName(protocol, cluster), pod)
}

// getPodDNSName returns the fully qualified DNS name for the given pod ID
func getPodDNSName(protocol *storagev2beta1.MultiRaftProtocol, cluster int, pod int) string {
	domain := os.Getenv(clusterDomainEnv)
	if domain == "" {
		domain = "cluster.local"
	}
	return fmt.Sprintf("%s-%d.%s.%s.svc.%s", getClusterName(protocol, cluster), pod, getClusterHeadlessServiceName(protocol, cluster), protocol.Namespace, domain)
}

// newClusterLabels returns the labels for the given cluster
func newClusterLabels(protocol *storagev2beta1.MultiRaftProtocol, cluster int) map[string]string {
	labels := make(map[string]string)
	for key, value := range protocol.Labels {
		labels[key] = value
	}
	labels[appLabel] = appAtomix
	labels[databaseLabel] = fmt.Sprintf("%s.%s", protocol.Name, protocol.Namespace)
	labels[clusterLabel] = fmt.Sprint(cluster)
	return labels
}

func getImage(storage *storagev2beta1.MultiRaftProtocol) string {
	if storage.Spec.Image != "" {
		return storage.Spec.Image
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
