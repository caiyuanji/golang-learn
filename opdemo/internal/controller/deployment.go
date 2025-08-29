package controller

import (
	"context"
	"fmt"
	appv1 "gitee.enflame.cn/ModelOps/opdemo/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewDeploy(app *appv1.WebService) *appsv1.Deployment {
	labels := map[string]string{"app": app.Name + "-" + app.Spec.Webapp.Name}
	selector := &metav1.LabelSelector{MatchLabels: labels}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,

			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:   appv1.GroupVersion.Group,
					Version: appv1.GroupVersion.Version,
					Kind:    "WebService",
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Webapp.Size,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: newContainers(app),
				},
			},
			Selector: selector,
		},
	}
}

func newContainers(app *appv1.WebService) []corev1.Container {
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range app.Spec.Webapp.Ports {
		cport := corev1.ContainerPort{}
		cport.ContainerPort = svcPort.TargetPort.IntVal
		containerPorts = append(containerPorts, cport)
	}
	return []corev1.Container{
		{
			Name:            app.Spec.Webapp.Name,
			Image:           app.Spec.Webapp.Image,
			Resources:       app.Spec.Webapp.Resources,
			Ports:           containerPorts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env:             app.Spec.Webapp.Envs,
		},
	}
}

func (r *WebServiceReconciler) mysqlAuthSecret(app *appv1.WebService) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "mysql-auth", Namespace: app.Namespace},
		Type:       "Opaque",
		StringData: map[string]string{"username": "demo", "password": "demo2024"},
	}
	controllerutil.SetControllerReference(app, secret, r.Scheme)
	return secret
}

func (r *WebServiceReconciler) mysqlDeployment(app *appv1.WebService) *appsv1.Deployment {
	labels := labels(app, "mysql")
	containerPort := app.Spec.Webapp.Ports[0].TargetPort.IntVal

	//int(app.Spec.Mysql.Ports[0].TargetPort)
	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "mysql-auth"}, Key: "username"},
	}

	passwordSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "mysql-auth"}, Key: "password"},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: app.Name + "-mysql", Namespace: app.Namespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Mysql.Size,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: app.Spec.Mysql.Image,
						Name:  app.Spec.Mysql.Name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: containerPort,
							Name:          "mysql",
						}},
						Env: []corev1.EnvVar{
							{Name: "MYSQL_ROOT_PASSWORD", Value: "password"},
							{Name: "MYSQL_DATABASE", Value: "webservice"},
							{Name: "MYSQL_USER", ValueFrom: userSecret},
							{Name: "MYSQL_PASSWORD", ValueFrom: passwordSecret},
						},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(app, dep, r.Scheme)
	return dep
}

// Returns whether or not the MySQL deployment is running
func (r *WebServiceReconciler) isMysqlUp(app *appv1.WebService) bool {

	deployment := &appsv1.Deployment{}

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: app.Name + "-mysql", Namespace: app.Namespace}, deployment)
	if err != nil {
		fmt.Println("Deployment mysql not found")
		return false
	}

	if deployment.Status.ReadyReplicas == 1 {
		return true
	}

	return false
}
