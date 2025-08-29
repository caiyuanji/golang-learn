/*
  @Author: yuanji.cai
  @Data: 2025/8/24 10:38
*/

package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	webappsv1 "my.domain/demo/api/v1"
)

func NewFrontendDeployment(webService *webappsv1.WebService) *appsv1.Deployment {
	myLabels := map[string]string{"app": webService.Spec.Frontend.Name}
	mySelector := &metav1.LabelSelector{MatchLabels: myLabels}
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: webService.Spec.Frontend.Name, Namespace: webService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(webService, schema.GroupVersionKind{
					Group:   webappsv1.GroupVersion.Group,
					Version: webappsv1.GroupVersion.Version,
					Kind:    "WebService",
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: webService.Spec.Frontend.Size,
			Selector: mySelector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: myLabels},
				Spec: corev1.PodSpec{
					Containers: newFrontendContainers(webService),
				},
			},
		},
	}

	return deployment
}

func newFrontendContainers(webService *webappsv1.WebService) []corev1.Container {
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range webService.Spec.Frontend.Ports {
		cPort := corev1.ContainerPort{}
		cPort.ContainerPort = svcPort.TargetPort.IntVal
		containerPorts = append(containerPorts, cPort)
	}

	containers := []corev1.Container{
		{
			Name:            "nginx",
			Image:           webService.Spec.Frontend.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Resources:       webService.Spec.Frontend.Resources,
			Env:             webService.Spec.Frontend.Envs,
			Ports:           containerPorts,
		},
	}
	return containers
}

func NewFrontendService(webService *webappsv1.WebService) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: webService.Spec.Frontend.Name, Namespace: webService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(webService, schema.GroupVersionKind{
					Group:   webappsv1.GroupVersion.Group,
					Version: webappsv1.GroupVersion.Version,
					Kind:    "WebService",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Ports:    webService.Spec.Frontend.Ports,
			Selector: map[string]string{"app": webService.Spec.Frontend.Name},
		},
	}

	return service
}
