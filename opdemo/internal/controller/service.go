package controller

import (
	appv1 "gitee.enflame.cn/ModelOps/opdemo/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewService(app *appv1.WebService) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Webapp.Name,
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:   appv1.GroupVersion.Group,
					Version: appv1.GroupVersion.Version,
					Kind:    "WebService",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeNodePort,
			Ports: app.Spec.Webapp.Ports,
			Selector: map[string]string{
				"app": app.Name + "-" + app.Spec.Webapp.Name,
			},
		},
	}
}

func (r *WebServiceReconciler) mysqlService(app *appv1.WebService) *corev1.Service {
	labels := labels(app, "mysql")
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Mysql.Name,
			Namespace: app.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports:    app.Spec.Mysql.Ports,
		},
	}

	controllerutil.SetControllerReference(app, s, r.Scheme)
	return s
}
