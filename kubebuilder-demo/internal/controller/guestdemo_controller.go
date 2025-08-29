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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	webappv1 "my.domain/demo/api/v1"
)

// GuestdemoReconciler reconciles a Guestdemo object
type GuestdemoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=webapp.my.domain,resources=guestdemoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.my.domain,resources=guestdemoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=webapp.my.domain,resources=guestdemoes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Guestdemo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *GuestdemoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here
	guestdemo := &webappv1.Guestdemo{}
	err := r.Get(ctx, req.NamespacedName, guestdemo)
	if err != nil {
		log.Println("Error:", err)
		return ctrl.Result{}, err
	}
	log.Println("guestdemo object:", guestdemo)

	if !guestdemo.DeletionTimestamp.IsZero() || len(guestdemo.Finalizers) > guestdemo.Spec.Num {
		err := r.ClearRedis(guestdemo, ctx)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	updateFlag := false
	redisPodNames := GetRedisPodName(guestdemo)
	for _, podName := range redisPodNames {
		finalizerPodName, err := CreateRedis(podName, r.Client, guestdemo, r.Scheme)
		if err != nil {
			log.Println("Create redis pod fail:", err)
			return ctrl.Result{}, err
		}

		//if finalizerPodName == "" {
		//	continue
		//}
		if IsContainString(guestdemo.Finalizers, podName) == false {
			guestdemo.Finalizers = append(guestdemo.Finalizers, finalizerPodName)
			updateFlag = true
		}
	}

	if updateFlag {
		err := r.Client.Update(ctx, guestdemo)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func IsContainString(sliceExam []string, str string) bool {
	for _, item := range sliceExam {
		if item == str {
			return true
		}
	}
	return false
}

func (r *GuestdemoReconciler) ClearRedis(guestdemo *webappv1.Guestdemo, ctx context.Context) error {
	deletePodNames := []string{}
	position := guestdemo.Spec.Num

	if len(guestdemo.Finalizers)-guestdemo.Spec.Num != 0 {
		deletePodNames = guestdemo.Finalizers[position:]
		guestdemo.Finalizers = guestdemo.Finalizers[:position]
	} else {
		deletePodNames = guestdemo.Finalizers[:]
		guestdemo.Finalizers = []string{}
	}

	for _, finalizerPodName := range deletePodNames {
		r.Client.Delete(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      finalizerPodName,
				Namespace: guestdemo.Namespace,
			},
		})
	}

	//guestdemo.Finalizers = []string{}
	return r.Client.Update(ctx, guestdemo)
}

// SetupWithManager sets up the controller with the Manager.
func (r *GuestdemoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Guestdemo{}).
		//Named("guestdemo").
		Owns(&corev1.Pod{}).
		Complete(r)
}
