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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	riov1 "qiniu.io/rio-csi/api/v1"
)

// VolumeReconciler reconciles a Volume object
type VolumeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=rio.qiniu.io,resources=volumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rio.qiniu.io,resources=volumes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rio.qiniu.io,resources=volumes/finalizers,verbs=update
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Volume object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *VolumeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	l.Info("hello" + req.Name + "||||" + req.Namespace)
	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&riov1.Volume{}).
		Complete(r)
}
