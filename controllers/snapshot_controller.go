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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	riov1 "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/logger"
	"qiniu.io/rio-csi/lvm"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SnapshotReconciler reconciles a Snapshot object
type SnapshotReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	NodeID string
}

//+kubebuilder:rbac:groups=rio.qiniu.io,resources=snapshots,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rio.qiniu.io,resources=snapshots/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rio.qiniu.io,resources=snapshots/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Snapshot object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *SnapshotReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger.StdLog.Infof("reconcile snapshot %s namespace %s", req.Name, req.Namespace)

	var snap riov1.Snapshot
	err := r.Get(ctx, client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, &snap)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.StdLog.Errorf("get snapshot %s error %v", req.Name, err)
	}

	if r.NodeID != snap.Spec.OwnerNodeID {
		return ctrl.Result{}, nil
	}

	err = r.syncSnapshot(ctx, &snap)
	if err != nil {
		logger.StdLog.Errorf("sync snapshot %s error %v", req.Name, err)
		// retry
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}

	return ctrl.Result{}, nil
}

func (r *SnapshotReconciler) syncSnapshot(ctx context.Context, snap *riov1.Snapshot) (err error) {
	// :remove
	// snapshot should be deleted. Check if deletion timestamp is set
	if snap.ObjectMeta.DeletionTimestamp != nil {
		err = lvm.DestroySnapshot(snap)
		if err != nil {
			_, err = crd.RemoveSnapFinalizer(snap)
		}

		r.Get()

		return err
	}

	// :create
	switch snap.Status.State {
	case crd.StatusFailed:
		logger.StdLog.Errorf("Skipping retrying lvm snapshot provisioning as its already in failed state: %s", snap.Name)
		return nil
	case crd.StatusReady:
		logger.StdLog.Infof("snapshot %s already provisioned", snap.Name)
		return nil
	}

	err = lvm.CreateSnapshot(snap)
	if err != nil {
		logger.StdLog.Error(err)
		_, err = crd.UpdateSnapInfo(snap, crd.StatusFailed)
		return err
	}

	_, err = crd.UpdateSnapInfo(snap, crd.StatusReady)
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *SnapshotReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&riov1.Snapshot{}).
		Complete(r)
}
