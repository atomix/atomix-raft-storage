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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	api "github.com/atomix/api/proto/atomix/controller"
	"github.com/atomix/kubernetes-controller/pkg/apis/cloud/v1beta3"
	"github.com/atomix/kubernetes-controller/pkg/controller/util/k8s"
	"github.com/atomix/raft-storage-controller/pkg/apis/storage/v1beta1"
	"github.com/gogo/protobuf/jsonpb"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("raft_storage_controller")

const (
	apiPort               = 5678
	protocolPort          = 5679
	probePort             = 5679
	defaultImage          = "atomix/raft-storage-node:latest"
	headlessServiceSuffix = "hs"
	clusterLabel          = "cluster"
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

var _ reconcile.Reconciler = &Reconciler{}

// Reconciler reconciles a Cluster object
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconcile Database")
	database := &v1beta3.Database{}
	err := r.client.Get(context.TODO(), request.NamespacedName, database)
	if err != nil {
		log.Error(err, "Reconcile Database")
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{Requeue: true}, err
	}

	log.Info("Reconcile RaftStorageClass")
	storage := &v1beta1.RaftStorageClass{}
	namespace := database.Spec.StorageClass.Namespace
	if namespace == "" {
		namespace = database.Namespace
	}
	name := types.NamespacedName{
		Namespace: namespace,
		Name:      database.Spec.StorageClass.Name,
	}
	err = r.client.Get(context.TODO(), name, storage)
	if err != nil {
		log.Error(err, "Reconcile RaftStorageClass")
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{Requeue: true}, err
	}

	log.Info("Reconcile Clusters")
	err = r.reconcileClusters(database, storage)
	if err != nil {
		log.Error(err, "Reconcile Clusters")
		return reconcile.Result{}, err
	}

	log.Info("Reconcile Partitions")
	err = r.reconcilePartitions(database, storage)
	if err != nil {
		log.Error(err, "Reconcile Partitions")
		return reconcile.Result{}, err
	}

	log.Info("Reconcile Status")
	err = r.reconcileStatus(database, storage)
	if err != nil {
		log.Error(err, "Reconcile Status")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileClusters(database *v1beta3.Database, storage *v1beta1.RaftStorageClass) error {
	clusters := getClusters(database, storage)
	for cluster := 1; cluster <= clusters; cluster++ {
		err := r.reconcileCluster(database, storage, cluster)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) reconcileCluster(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) error {
	err := r.reconcileConfigMap(database, storage, cluster)
	if err != nil {
		return err
	}

	err = r.reconcileStatefulSet(database, storage, cluster)
	if err != nil {
		return err
	}

	err = r.reconcileService(database, storage, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) reconcileConfigMap(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) error {
	log.Info("Reconcile raft storage config map")
	cm := &corev1.ConfigMap{}
	name := types.NamespacedName{
		Namespace: database.Namespace,
		Name:      getClusterName(database, cluster),
	}
	err := r.client.Get(context.TODO(), name, cm)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addConfigMap(database, storage, cluster)
	}
	return err
}

func (r *Reconciler) addConfigMap(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) error {
	log.Info("Creating raft ConfigMap", "Name", database.Name, "Namespace", database.Namespace)
	var config interface{}

	clusterConfig, err := newNodeConfigString(database, storage, cluster)
	if err != nil {
		return err
	}

	protocolConfig, err := newProtocolConfigString(config)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getClusterName(database, cluster),
			Namespace: database.Namespace,
			Labels:    newClusterLabels(database, cluster),
		},
		Data: map[string]string{
			clusterConfigFile:  clusterConfig,
			protocolConfigFile: protocolConfig,
		},
	}

	if err := controllerutil.SetControllerReference(database, cm, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), cm)
}

// newNodeConfigString creates a node configuration string for the given cluster
func newNodeConfigString(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) (string, error) {
	members := make([]*api.MemberConfig, storage.Spec.Replicas)
	for i := 0; i < int(storage.Spec.Replicas); i++ {
		members[i] = &api.MemberConfig{
			ID:           getPodName(database, cluster, i),
			Host:         getPodDNSName(database, cluster, i),
			ProtocolPort: protocolPort,
			APIPort:      apiPort,
		}
	}

	partitions := make([]*api.PartitionId, 0, database.Spec.Partitions)
	for partitionID := 1; partitionID <= int(database.Spec.Partitions); partitionID++ {
		if getClusterForPartitionID(database, storage, partitionID) == cluster {
			partition := &api.PartitionId{
				Partition: int32(partitionID),
				Cluster: &api.ClusterId{
					ID: int32(cluster),
					DatabaseID: &api.DatabaseId{
						Name:      database.Name,
						Namespace: database.Namespace,
					},
				},
			}
			partitions = append(partitions, partition)
		}
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

func (r *Reconciler) reconcileStatefulSet(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) error {
	log.Info("Reconcile raft storage stateful set")
	statefulSet := &appsv1.StatefulSet{}
	name := types.NamespacedName{
		Namespace: database.Namespace,
		Name:      getClusterName(database, cluster),
	}
	err := r.client.Get(context.TODO(), name, statefulSet)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addStatefulSet(database, storage, cluster)
	}
	return err
}

func newDataVolume(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) corev1.Volume {
	dataVolume := corev1.Volume{
		Name: dataVolume,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	return dataVolume

}

func newConfigVolume(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) corev1.Volume {
	configVolume := corev1.Volume{

		Name: configVolume,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: getClusterName(database, cluster),
				},
			},
		},
	}
	return configVolume

}

func (r *Reconciler) addStatefulSet(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) error {
	log.Info("Creating raft replicas", "Name", database.Name, "Namespace", database.Namespace)

	image := storage.Spec.Image
	if image == "" {
		image = defaultImage
	}
	pullPolicy := storage.Spec.ImagePullPolicy
	if pullPolicy == "" {
		pullPolicy = corev1.PullIfNotPresent
	}

	volumes := []corev1.Volume{
		newConfigVolume(database, storage, cluster),
	}

	log.Info("volume Claim:", "template", storage.Spec)
	log.Info("image", "name", storage.Spec.Image)

	if storage.Spec.VolumeClaimTemplates == nil {
		log.Info("here")
		volumes = append(volumes, newDataVolume(database, storage, cluster))
	}

	set := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getClusterName(database, cluster),
			Namespace: database.Namespace,
			Labels:    newClusterLabels(database, cluster),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: getClusterHeadlessServiceName(database, cluster),
			Replicas:    &storage.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: newClusterLabels(database, cluster),
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			PodManagementPolicy:  appsv1.ParallelPodManagement,
			VolumeClaimTemplates: storage.Spec.VolumeClaimTemplates,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: newClusterLabels(database, cluster),
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
									Name:      dataVolume,
									MountPath: dataPath,
								},
								{
									Name:      configVolume,
									MountPath: configPath,
								},
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(database, set, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), set)
}

func (r *Reconciler) reconcileService(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) error {
	log.Info("Reconcile raft storage headless service")
	service := &corev1.Service{}
	name := types.NamespacedName{
		Namespace: database.Namespace,
		Name:      getClusterHeadlessServiceName(database, cluster),
	}
	err := r.client.Get(context.TODO(), name, service)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addService(database, storage, cluster)
	}
	return err
}

func (r *Reconciler) addService(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, cluster int) error {
	log.Info("Creating headless raft service", "Name", database.Name, "Namespace", database.Namespace)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getClusterHeadlessServiceName(database, cluster),
			Namespace: database.Namespace,
			Labels:    newClusterLabels(database, cluster),
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
			Selector:                 database.Labels,
		},
	}

	if err := controllerutil.SetControllerReference(database, service, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), service)
}

func (r *Reconciler) reconcilePartitions(database *v1beta3.Database, storage *v1beta1.RaftStorageClass) error {
	options := &client.ListOptions{
		Namespace:     database.Namespace,
		LabelSelector: labels.SelectorFromSet(k8s.GetPartitionLabelsForDatabase(database)),
	}
	partitions := &v1beta3.PartitionList{}
	err := r.client.List(context.TODO(), partitions, options)
	if err != nil {
		return err
	}

	for _, partition := range partitions.Items {
		err := r.reconcilePartition(database, storage, partition)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) reconcilePartition(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, partition v1beta3.Partition) error {
	service := &corev1.Service{}
	name := types.NamespacedName{
		Namespace: partition.Namespace,
		Name:      partition.Spec.ServiceName,
	}
	err := r.client.Get(context.TODO(), name, service)
	if err == nil || !k8serrors.IsNotFound(err) {
		return err
	}

	cluster := getClusterForPartition(database, storage, &partition)
	service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   partition.Namespace,
			Name:        partition.Spec.ServiceName,
			Labels:      partition.Labels,
			Annotations: partition.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector: newClusterLabels(database, cluster),
			Ports: []corev1.ServicePort{
				{
					Name: "api",
					Port: 5678,
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(&partition, service, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), service)
}

func (r *Reconciler) reconcileStatus(database *v1beta3.Database, storage *v1beta1.RaftStorageClass) error {
	options := &client.ListOptions{
		Namespace:     database.Namespace,
		LabelSelector: labels.SelectorFromSet(k8s.GetPartitionLabelsForDatabase(database)),
	}
	partitions := &v1beta3.PartitionList{}
	err := r.client.List(context.TODO(), partitions, options)
	if err != nil {
		return err
	}

	for _, partition := range partitions.Items {
		if !partition.Status.Ready {
			log.Info("Reconcile status", "Database", database.Name, "Partition", partition.Name, "Ready", partition.Status.Ready)
			cluster := getClusterForPartition(database, storage, &partition)

			statefulSet := &appsv1.StatefulSet{}
			name := types.NamespacedName{
				Namespace: getClusterName(database, cluster),
				Name:      database.Name,
			}
			err := r.client.Get(context.TODO(), name, statefulSet)
			if err != nil {
				if k8serrors.IsNotFound(err) {
					return nil
				}
				return err
			}

			if statefulSet.Status.ReadyReplicas == statefulSet.Status.Replicas {
				partition.Status.Ready = true
				log.Info("Updating Partition status", "Name", partition.Name, "Namespace", partition.Namespace, "Ready", partition.Status.Ready)
				err = r.client.Status().Update(context.TODO(), &partition)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// getClusters returns the number of clusters in the given database
func getClusters(database *v1beta3.Database, storage *v1beta1.RaftStorageClass) int {
	partitions := database.Spec.Partitions
	if partitions == 0 {
		partitions = 1
	}
	partitionsPerCluster := storage.Spec.PartitionsPerCluster
	if partitionsPerCluster == 0 {
		partitionsPerCluster = 1
	}
	return int(math.Ceil(float64(partitions) / float64(partitionsPerCluster)))
}

// getClusterForPartition returns the cluster ID for the given partition ID
func getClusterForPartition(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, partition *v1beta3.Partition) int {
	return getClusterForPartitionID(database, storage, int(partition.Spec.PartitionID))
}

// getClusterForPartitionID returns the cluster ID for the given partition ID
func getClusterForPartitionID(database *v1beta3.Database, storage *v1beta1.RaftStorageClass, partition int) int {
	return (partition % getClusters(database, storage)) + 1
}

// getClusterResourceName returns the given resource name for the given cluster
func getClusterResourceName(database *v1beta3.Database, cluster int, resource string) string {
	return fmt.Sprintf("%s-%s", getClusterName(database, cluster), resource)
}

// getClusterName returns the cluster name
func getClusterName(database *v1beta3.Database, cluster int) string {
	return fmt.Sprintf("%s-%d", database.Name, cluster)
}

// getClusterHeadlessServiceName returns the headless service name for the given cluster
func getClusterHeadlessServiceName(database *v1beta3.Database, cluster int) string {
	return getClusterResourceName(database, cluster, headlessServiceSuffix)
}

// getPodName returns the name of the pod for the given pod ID
func getPodName(database *v1beta3.Database, cluster int, pod int) string {
	return fmt.Sprintf("%s-%d", getClusterName(database, cluster), pod)
}

// getPodDNSName returns the fully qualified DNS name for the given pod ID
func getPodDNSName(database *v1beta3.Database, cluster int, pod int) string {
	return fmt.Sprintf("%s-%d.%s.%s.svc.cluster.local", getClusterName(database, cluster), pod, getClusterHeadlessServiceName(database, cluster), database.Namespace)
}

// newClusterLabels returns the labels for the given cluster
func newClusterLabels(database *v1beta3.Database, cluster int) map[string]string {
	labels := make(map[string]string)
	for key, value := range database.Labels {
		labels[key] = value
	}
	labels[clusterLabel] = fmt.Sprint(cluster)
	return labels
}
