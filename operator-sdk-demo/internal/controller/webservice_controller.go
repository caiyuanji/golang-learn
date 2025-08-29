/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"my.domain/demo/internal/resources"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"log"
	webappsv1 "my.domain/demo/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// WebServiceReconciler reconciles a WebService object
type WebServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=webapps.my.domain,resources=webservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapps.my.domain,resources=webservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=webapps.my.domain,resources=webservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WebService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *WebServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = ctrllog.FromContext(ctx)

	// TODO(user): your logic here
	webService := &webappsv1.WebService{}
	err := r.Client.Get(ctx, req.NamespacedName, webService)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	log.Println("WebService object:", webService)

	if webService.DeletionTimestamp != nil {
		log.Println("WebService will be delete.")
		return ctrl.Result{}, nil
	}

	r.ensureDBSecret(webService, r.mysqlSecret(webService))
	r.ensureDBDeployment(webService, r.mysqlDeployment(webService))
	r.ensureDBService(webService, r.mysqlService(webService))

	running := r.isMysqlRunning(webService)
	if !running {
		delay := time.Second * time.Duration(5)
		return ctrl.Result{RequeueAfter: delay}, err
	}
	log.Println("Mysql pod is running...")

	deploy := &appsv1.Deployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: webService.Namespace, Name: webService.Spec.Frontend.Name}, deploy)
	if err != nil && errors.IsNotFound(err) {
		log.Println("Frontend deployment not found.")

		if err := r.Client.Create(ctx, resources.NewFrontendDeployment(webService)); err != nil {
			log.Println("Frontend deployment create failure.")
			return ctrl.Result{}, err
		}
		log.Println("Frontend deployment create success.")

		if err := r.Client.Create(ctx, resources.NewFrontendService(webService)); err != nil {
			log.Println("Frontend service create failure.")
			return ctrl.Result{}, err
		}
		log.Println("Frontend service create success.")

		specData, _ := json.Marshal(webService.Spec)
		if webService.Annotations != nil {
			webService.Annotations["spec"] = string(specData)
		} else {
			webService.Annotations = map[string]string{"spec": string(specData)}
		}
		if err := r.Client.Update(ctx, webService); err != nil {
			log.Println("webService update failure.")
			return ctrl.Result{}, err
		}
		log.Println("webService update success.")

		return ctrl.Result{}, nil
	}

	oldSpec := webappsv1.WebServiceSpec{}
	if err := json.Unmarshal([]byte(webService.Annotations["spec"]), &oldSpec); err != nil {
		log.Println("Annotations['spec'] unmarshal failure.")
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(webService.Spec, oldSpec) {
		log.Println("webService necessary update.")

		newFrontendDeploy := resources.NewFrontendDeployment(webService)
		oldFrontendDeploy := &appsv1.Deployment{}
		r.Client.Get(ctx, types.NamespacedName{Name: webService.Spec.Frontend.Name, Namespace: webService.Namespace}, oldFrontendDeploy)
		oldFrontendDeploy.Spec = newFrontendDeploy.Spec
		if err := r.Update(ctx, oldFrontendDeploy); err != nil {
			log.Println("Frontend deployment update failure.")
			return ctrl.Result{}, err
		}
		log.Println("Frontend deployment update success.")

		newFrontendSvc := resources.NewFrontendService(webService)
		oldFrontendSvc := &corev1.Service{}
		r.Client.Get(ctx, types.NamespacedName{Name: webService.Spec.Frontend.Name, Namespace: webService.Namespace}, oldFrontendSvc)
		newFrontendSvc.Spec.ClusterIP = oldFrontendSvc.Spec.ClusterIP
		oldFrontendSvc.Spec = newFrontendSvc.Spec
		if err := r.Update(ctx, oldFrontendSvc); err != nil {
			log.Println("Frontend service update failure.")
			return ctrl.Result{}, err
		}
		log.Println("Frontend service update success.")

		specData, _ := json.Marshal(webService.Spec)
		if webService.Annotations != nil {
			webService.Annotations["spec"] = string(specData)
		} else {
			webService.Annotations = map[string]string{"spec": string(specData)}
		}
		if err := r.Client.Update(ctx, webService); err != nil {
			log.Println("webService update failure.")
			return ctrl.Result{}, err
		}
		log.Println("webService update success.")

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappsv1.WebService{}).
		Complete(r)
}
