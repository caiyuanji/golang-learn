/*
  @Author: yuanji.cai
  @Data: 2025/8/14 15:08
*/

package controller

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	webappv1 "my.domain/demo/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func CreateRedis(podName string, client client.Client, guestdemo *webappv1.Guestdemo, scheme *runtime.Scheme) (string, error) {
	if IsExists(podName, guestdemo, client) {
		return podName, nil
	}
	newPod := &corev1.Pod{}
	newPod.Name = podName
	newPod.Namespace = guestdemo.Namespace
	newPod.Spec.Containers = []corev1.Container{
		{
			Name:            guestdemo.Name,
			Image:           "xci-harbor.enflame.cn/docker.io/library/redis:5-alpine",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: int32(guestdemo.Spec.Port),
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(guestdemo, newPod, scheme)
	if err != nil {
		return podName, err
	}
	return podName, client.Create(context.Background(), newPod)
}

func IsExists(podName string, guestdemo *webappv1.Guestdemo, client client.Client) bool {
	//for _, finalizerPodName := range guestdemo.Finalizers {
	//	if podName == finalizerPodName {
	//		return true
	//	}
	//}
	//return false

	err := client.Get(context.Background(), types.NamespacedName{Name: podName, Namespace: guestdemo.Namespace}, &corev1.Pod{})
	if err != nil {
		return false
	}
	return true
}

func GetRedisPodName(guestdemo *webappv1.Guestdemo) []string {
	redisPodNames := make([]string, guestdemo.Spec.Num)
	for i := 0; i < guestdemo.Spec.Num; i++ {
		redisPodNames[i] = fmt.Sprintf("%s-%d", guestdemo.Name, i)
	}
	return redisPodNames
}
