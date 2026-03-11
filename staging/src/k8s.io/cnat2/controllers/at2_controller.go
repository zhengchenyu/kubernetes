/*
Copyright 2021 zhengchenyu.

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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cnat2v1alpha1 "k8s.io/kubernetes/api/v1alpha1"
)

// At2Reconciler reconciles a At2 object
type At2Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cnat2.example.org,resources=at2s;pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cnat2.example.org,resources=at2s/status;pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cnat2.example.org,resources=at2s/finalizers;pods/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=at2s;pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=at2s/status;pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=at2s/finalizers;pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the At2 object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *At2Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx, "namesapce", req.Namespace, "at1", req.Name)

	// your logic here
	reqLogger.Info("Reconcile start ... ")

	instance := &cnat2v1alpha1.At2{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("at1 not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if instance.Status.Phase == "" {
		instance.Status.Phase = cnat2v1alpha1.PhasePending
	}

	switch instance.Status.Phase {
	case cnat2v1alpha1.PhasePending:
		reqLogger.Info("instace is PhasePending")
		reqLogger.Info("instance checking schedule", "Scheduler", instance.Spec.Scheduler)
		d, err := timeUntilSchedule(instance.Spec.Scheduler, reqLogger)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("schedule parsing failed: %v", err))
			// Error reading the schedule - requeue the request:
			return ctrl.Result{}, err
		}
		reqLogger.Info("schedule parsing done.", "diff", d)
		if d > 0 {
			// Not yet time to execute the command, wait until the scheduled time
			return ctrl.Result{RequeueAfter: d}, nil
		}
		reqLogger.Info("it's time! Ready to execute.", "command", instance.Spec.Command)
		instance.Status.Phase = cnat2v1alpha1.PhaseRunning
	case cnat2v1alpha1.PhaseRunning:
		reqLogger.Info("Phase: RUNNING")
		pod := newPodForCR(instance)

		controllerutil.SetControllerReference(instance, pod, r.Scheme)

		found := &corev1.Pod{}
		// nsName := types.NamespacedName{Namespace: req.Namespace, Name: req.Name}
		err := r.Get(context.TODO(), req.NamespacedName, found)

		if err != nil && errors.IsNotFound(err) {
			err := r.Create(context.TODO(), pod)
			if err != nil {
				return ctrl.Result{}, err
			}
			reqLogger.Info("pod launched")
		} else if err != nil {
			return ctrl.Result{}, err
		} else if found.Status.Phase == corev1.PodFailed || found.Status.Phase == corev1.PodSucceeded {
			reqLogger.Info("Container Terminated", "reason", found.Status.Reason,
				"message", found.Status.Message)
			instance.Status.Phase = cnat2v1alpha1.PhaseDone
		} else {
			return ctrl.Result{}, nil
		}

	case cnat2v1alpha1.PhaseDone:
		reqLogger.Info("phase: DONE")
		return ctrl.Result{}, nil
	default:
		reqLogger.Info("NOP")
		return ctrl.Result{}, nil
	}

	err = r.Status().Update(context.TODO(), instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func newPodForCR(cr *cnat2v1alpha1.At2) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": cr.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    cr.Spec.Name,
					Image:   cr.Spec.Image,
					Command: strings.Split(cr.Spec.Command, " "),
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
	}
}

func timeUntilSchedule(schedule string, logger logr.Logger) (time.Duration, error) {
	now := time.Now().UTC()
	layout := "2006-01-02T15:04:05Z"
	s, err := time.Parse(layout, schedule)
	if err != nil {
		logger.Info("parse failed", "scheduler", schedule, "error", err)
		return time.Duration(0), nil
	}
	return s.Sub(now), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *At2Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cnat2v1alpha1.At2{}).
		Complete(r)
}
