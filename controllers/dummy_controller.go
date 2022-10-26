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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DummyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx) //.WithValues("Dummy", req.NamespacedName)

	// Lookup the Dummy instance
	dummy := &interviewcomv1alpha1.Dummy{}

	// Check if the CR must be deleted
	err := r.Get(ctx, req.NamespacedName, dummy)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Dummy resource not found. Nothing to be deleted")
			return ctrl.Result{}, nil
		} else { // Error reading the object - requeue the request.
			log.Error(err, "Failed to get Dummy")
			return ctrl.Result{}, err
		}

	}

	log.Info("Creating a new Dummy", "name: ", dummy.Name, "namespace: ", dummy.Namespace, "message: ", dummy.Spec.Message)

	// Copy the value of spec.message into status.specEcho
	dummy.Status.SpecEcho = dummy.Spec.Message
	err = r.Status().Update(ctx, dummy)
	if err != nil {
		log.Error(err, "Failed to update Dummy status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DummyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&interviewcomv1alpha1.Dummy{}).
		Complete(r)
}
