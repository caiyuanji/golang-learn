/*
  @Author: yuanji.cai
  @Data: 2025/8/24 12:37
*/

package controller

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	webappsv1 "my.domain/demo/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *WebServiceReconciler) mysqlSecret(webSerivce *webappsv1.WebService) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "mysql-auth", Namespace: webSerivce.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"username": "demo", "password": "demo2025"},
	}
	controllerutil.SetControllerReference(webSerivce, secret, r.Scheme)
	return secret
}

func (r *WebServiceReconciler) mysqlDeployment(webSerivce *webappsv1.WebService) *appsv1.Deployment {
	myLabels := makeLabels(webSerivce, "mysql")
	mySelector := &metav1.LabelSelector{MatchLabels: myLabels}
	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "mysql-auth"},
			Key:                  "username",
		},
	}
	passwordSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "mysql-auth"},
			Key:                  "password",
		},
	}
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: webSerivce.Spec.Mysql.Name, Namespace: webSerivce.Namespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: webSerivce.Spec.Mysql.Size,
			Selector: mySelector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: myLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            webSerivce.Spec.Mysql.Name,
							Image:           webSerivce.Spec.Mysql.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources:       webSerivce.Spec.Mysql.Resources,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: webSerivce.Spec.Mysql.Ports[0].TargetPort.IntVal,
									Name:          "mysql",
								},
							},
							Env: []corev1.EnvVar{
								{Name: "MYSQL_ROOT_PASSWORD", Value: "password"},
								{Name: "MYSQL_DATABASE", Value: "webservice"},
								{Name: "username", ValueFrom: userSecret},
								{Name: "password", ValueFrom: passwordSecret},
							},
						},
					},
				},
			},
		},
	}
	controllerutil.SetControllerReference(webSerivce, deploy, r.Scheme)
	return deploy
}

func (r *WebServiceReconciler) mysqlService(webSerivce *webappsv1.WebService) *corev1.Service {
	myLabels := makeLabels(webSerivce, "mysql")
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: webSerivce.Spec.Mysql.Name, Namespace: webSerivce.Namespace},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    webSerivce.Spec.Mysql.Ports,
			Selector: myLabels,
		},
	}
	controllerutil.SetControllerReference(webSerivce, svc, r.Scheme)
	return svc
}

func makeLabels(webService *webappsv1.WebService, tier string) map[string]string {
	labels := map[string]string{
		"app":           "webservice",
		"tier":          tier,
		"webservice-cr": webService.Name,
	}
	return labels
}

func (r *WebServiceReconciler) ensureDBSecret(webSerivce *webappsv1.WebService, secret *corev1.Secret) error {
	founder := &corev1.Secret{}
	err := r.Client.Get(context.Background(), types.NamespacedName{Name: "mysql-auth", Namespace: webSerivce.Namespace}, founder)
	if err != nil && errors.IsNotFound(err) {
		if err := r.Client.Create(context.Background(), secret); err != nil {
			log.Println("Mysql secret create failure.")
			return err
		} else {
			log.Println("Mysql secret create success.")
			return nil
		}
	}
	return nil
}

func (r *WebServiceReconciler) ensureDBDeployment(webSerivce *webappsv1.WebService, deploy *appsv1.Deployment) error {
	founder := &appsv1.Deployment{}
	err := r.Client.Get(context.Background(), types.NamespacedName{Name: webSerivce.Spec.Mysql.Name, Namespace: webSerivce.Namespace}, founder)
	if err != nil && errors.IsNotFound(err) {
		if err := r.Client.Create(context.Background(), deploy); err != nil {
			log.Println("Mysql deployment create failure.")
			return err
		} else {
			log.Println("Mysql deployment create success.")
			return nil
		}
	}
	return nil
}

func (r *WebServiceReconciler) ensureDBService(webSerivce *webappsv1.WebService, service *corev1.Service) error {
	founder := &corev1.Service{}
	err := r.Client.Get(context.Background(), types.NamespacedName{Name: webSerivce.Spec.Mysql.Name, Namespace: webSerivce.Namespace}, founder)
	if err != nil && errors.IsNotFound(err) {
		if err := r.Client.Create(context.Background(), service); err != nil {
			log.Println("Mysql service create failure.")
			return err
		} else {
			log.Println("Mysql service create success.")
			return nil
		}
	}
	return nil
}

func (r *WebServiceReconciler) isMysqlRunning(webService *webappsv1.WebService) bool {
	dbDeployment := &appsv1.Deployment{}
	err := r.Client.Get(context.Background(), types.NamespacedName{Name: webService.Spec.Mysql.Name, Namespace: webService.Namespace}, dbDeployment)
	if err != nil {
		return false
	}
	if dbDeployment.Status.ReadyReplicas > 0 {
		return true
	}
	return false
}
