/*
Copyright 2024.

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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// EventsReconciler reconciles a Events object
type EventsReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=hjort.uk,resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=hjort.uk,resources=events/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=hjort.uk,resources=events/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Events object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *EventsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO(user): your logic here
	jobThatTriggeredReconcile := &batchv1.Job{}
	err := r.Get(ctx, req.NamespacedName, jobThatTriggeredReconcile)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			logger.Info("pod resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get memcached")
		return ctrl.Result{}, err
	}
	// Test that I can get labels
	labels := jobThatTriggeredReconcile.GetLabels()
	if _, ok := labels["remote-app"]; !ok {
		// Key exists in myMap, and value is the corresponding value
		return ctrl.Result{}, nil
	}

	spec := jobThatTriggeredReconcile.Spec.Template.Spec
	nodeName := spec.NodeName
	if nodeName == "" {
		logger.Info("Spec does not have nodeName")
		return ctrl.Result{}, nil
	}
	// Listing all nodes in the cluster
	// Would this be expensive to do all the time
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList); err != nil {
		logger.Error(err, "Failed to list nodes")
		return ctrl.Result{}, err
	}
	for _, node := range nodeList.Items {
		if node.Name == nodeName {
			logger.Info("Node", "Name", node.Name)
			return ctrl.Result{}, nil
		}
	}

	r.Recorder.Event(jobThatTriggeredReconcile, "Warning", "NodeCheckFailed", "Job has specified a nodeName that does not exist")

	logger.Info("Haha")

	return ctrl.Result{}, nil
}

func ignoreDeletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			_, exists := e.Object.GetLabels()["remote-app"]
			return exists
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&batchv1.Job{}).WithEventFilter(ignoreDeletionPredicate()).
		Complete(r)
}
