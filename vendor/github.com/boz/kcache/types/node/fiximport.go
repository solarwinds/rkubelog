package node

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ metav1.Object
var _ corev1.Pod
var _ corev1.Secret
var _ corev1.Service
var _ corev1.Event
var _ corev1.Node
var _ corev1.ReplicationController
var _ appsv1.Deployment
var _ networkingv1beta1.Ingress
var _ appsv1.ReplicaSet
var _ appsv1.DaemonSet
var _ batchv1.Job
var _ appsv1.StatefulSet
