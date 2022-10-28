/*
Copyright 2022.

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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/ilmavridis/dummy-operator/api/v1alpha1"
	interviewcomv1alpha1 "github.com/ilmavridis/dummy-operator/api/v1alpha1"
)

// DummyReconciler reconciles a Dummy object
type DummyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=interview.com,resources=dummies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=interview.com,resources=dummies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=interview.com,resources=dummies/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DummyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Lookup the Dummy instance
	dummy := &interviewcomv1alpha1.Dummy{}
	err := r.Get(ctx, req.NamespacedName, dummy)

	existingPod := &corev1.Pod{}
	podImage := "nginx"

	// Check if the Dummy should be deleted
	if err != nil {
		if errors.IsNotFound(err) { // Dummy resource not found. It's ok since the dummy object should be deleted

			// In case there is no parent/child relationship established between the Dummy and its Pod,
			// the Pod will not be automatically deleted by K8s when its Dummy is deleted.
			// Check if there is a Pod to delete
			err := r.Get(ctx, req.NamespacedName, existingPod)
			if err != nil {
				if errors.IsNotFound(err) { // No Pod found, no need to do anything
					return ctrl.Result{}, nil
				} else {
					log.Error(err, "Failed to get Pod")
					return ctrl.Result{}, err
				}
			} else {
				r.Delete(ctx, existingPod) // Pod found, delete it
				if existingPod.DeletionTimestamp != nil {
					log.Info("Dummy not found, its Pod is deleted")
				}
			}

		} else {
			log.Error(err, "Failed to get Dummy")
			return ctrl.Result{}, err
		}

	} else {

		dummyName := dummy.Name
		dummyNamespace := dummy.Namespace
		dummyMessage := dummy.Spec.Message

		if dummy.Status.PodStatus == "Running" {
			log.Info("A Dummy and its Pod have been successfully deployed", "name", dummyName, "namespace", dummyNamespace, "message", dummyMessage)
		}

		// Copy the value of spec.message to status.specEcho
		dummy.Status.SpecEcho = dummyMessage
		err = r.Status().Update(ctx, dummy)
		if err != nil {
			log.Error(err, "Failed to update Dummy status")
			return ctrl.Result{}, err
		}

		// Check if the Pod already exists. If not, create a new one
		existingPod := &corev1.Pod{}
		err = r.Get(ctx, types.NamespacedName{Name: dummyName, Namespace: dummyNamespace}, existingPod)
		if err != nil && errors.IsNotFound(err) {
			// Define a new Pod
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: dummyNamespace,
					Name:      dummyName,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "nginx",
							Name:  "nginx",
						},
					},
				},
			}
			// Establish the relationship between the Dummy and its Pod
			err = ctrl.SetControllerReference(dummy, pod, r.Scheme)
			if err != nil {
				log.Error(err, "Establish the relationship between the Dummy and its Pod", "Pod name", pod.Name, "Pod namespace", pod.Namespace)
				return ctrl.Result{}, err
			}

			err = r.Create(ctx, pod)
			if err != nil {
				log.Error(err, "Failed to create a new Pod", "Pod name", pod.Name, "Pod namespace", pod.Namespace)
				return ctrl.Result{}, err
			}

		} else if err == nil { // Pod already exists

			// If the Pod's status is changed, update the Dummy's PodStatus field accordingly
			if string(existingPod.Status.Phase) != dummy.Status.PodStatus {
				dummy.Status.PodStatus = string(existingPod.Status.Phase)
				err = r.Status().Update(ctx, dummy)
				if err != nil {
					log.Error(err, "Failed to update Pod status")
					return ctrl.Result{}, err
				}

				if existingPod.Spec.Containers[0].Image == podImage {
					log.Info("A Dummy has been successfully deployed.", "its Pod is", existingPod.Status.Phase)
				}

			} else if err != nil {
				log.Error(err, "Failed to get Pod")
				return ctrl.Result{}, err
			}

			// Check if the Pod image is correct. If not, it needs to be updated
			if existingPod.Spec.Containers[0].Image != podImage {
				log.Info("Update Pod's image")
				existingPod.Spec.Containers[0].Image = podImage
				err = r.Update(ctx, existingPod)
				if err != nil {
					log.Error(err, "Failed to update Pod's image")
					return ctrl.Result{}, err
				}
			}

			// If there are more than one container in the Pod, delete it
			if len(existingPod.Spec.Containers) > 1 {
				log.Info("There are more than 1 container in this Pod. The Pod will be deleted", "Pod name", existingPod.Name, "Pod namespace", existingPod.Namespace)
				r.Delete(ctx, existingPod)
			}

		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DummyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&interviewcomv1alpha1.Dummy{}).
		// Check for changes in the associated Pod
		Watches(&source.Kind{Type: &corev1.Pod{}},
			&handler.EnqueueRequestForOwner{
				IsController: true,
				OwnerType:    &v1alpha1.Dummy{}}).
		// Check for delete events for a Pod. This is useful if there is no relationship between a Dummy and its Pod
		Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return false
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return true
			},
		})).
		Complete(r)
}
