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
	"fmt"
	riov1 "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/lvm"
	"regexp"
	"sort"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VolumeReconciler reconciles a Volume object
type VolumeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	NodeID string
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
	var vol riov1.Volume
	l.Info("hello" + req.Name + "||||" + req.Namespace)

	// TODO(user): your logic here
	err := r.Get(ctx, client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, &vol)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if r.NodeID != vol.Spec.OwnerNodeID {
		l.Error(fmt.Errorf("nodeid: %s, vol node id %s is not same", r.NodeID, vol.Spec.OwnerNodeID), "vol is not on this node")
		return ctrl.Result{}, nil
	}

	err = r.syncVol(ctx, &vol)
	if err != nil {
		l.Error(err, "sync vol error")
	}

	return ctrl.Result{}, nil
}

func (r *VolumeReconciler) syncVol(ctx context.Context, vol *riov1.Volume) error {
	l := log.FromContext(ctx)
	var err error
	// :remove
	// LVM Volume should be deleted. Check if deletion timestamp is set
	if r.isDeletionCandidate(vol) {
		err = lvm.DestroyVolume(vol)
		if err == nil {
			err = lvm.RemoveVolFinalizer(vol)
		}
		return err
	}

	// if status is Pending then it means we are creating the volume.
	// Otherwise, we are just ignoring the event.
	switch vol.Status.State {
	case lvm.LVMStatusFailed:
		l.Error(nil, "Skipping retrying lvm volume provisioning as its already in failed state: %+v", vol.Status.Error)
		return nil
	case lvm.LVMStatusReady:
		l.Info("lvm volume already provisioned")
		return nil
	}

	// if there is already a volGroup field set for lvmvolume resource,
	// we'll first try to create a volume in that volume group.
	if vol.Spec.VolGroup != "" {
		err = lvm.CreateVolume(vol)
		if err == nil {
			return lvm.UpdateVolInfo(vol, lvm.LVMStatusReady)
		}
	}

	vgs, err := r.getVgPriorityList(vol)
	if err != nil {
		return err
	}

	if len(vgs) == 0 {
		err = fmt.Errorf("no vg available to serve volume request having regex=%q & capacity=%q",
			vol.Spec.VgPattern, vol.Spec.Capacity)
		l.Error(nil, fmt.Sprintf("lvm volume %v - %v", vol.Name, err))
	} else {
		for _, vg := range vgs {
			// first update volGroup field in lvm volume resource for ensuring
			// idempotency and avoiding volume leaks during crash.
			if vol, err = lvm.UpdateVolGroup(vol, vg.Name); err != nil {
				l.Error(nil, fmt.Sprintf("failed to update volGroup to %v: %v", vg.Name, err))
				return err
			}
			if err = lvm.CreateVolume(vol); err == nil {
				return lvm.UpdateVolInfo(vol, lvm.LVMStatusReady)
			}
		}
	}

	// In case no vg available or lvm.CreateVolume fails for all vgs, mark
	// the volume provisioning failed so that controller can reschedule it.
	vol.Status.Error = r.transformLVMError(err)
	return lvm.UpdateVolInfo(vol, lvm.LVMStatusFailed)
}

func (r *VolumeReconciler) transformLVMError(err error) *riov1.VolumeError {
	volErr := &riov1.VolumeError{
		Code:    riov1.Internal,
		Message: err.Error(),
	}
	execErr, ok := err.(*lvm.ExecError)
	if !ok {
		return volErr
	}

	if strings.Contains(strings.ToLower(string(execErr.Output)),
		"insufficient free space") {
		volErr.Code = riov1.InsufficientCapacity
	}
	return volErr
}

// getVgPriorityList returns ordered list of volume groups from higher to lower
// priority to use for provisioning a lvm volume. As of now, we are prioritizing
// the vg having least amount free space available to fit the volume.
func (r *VolumeReconciler) getVgPriorityList(vol *riov1.Volume) ([]riov1.VolumeGroup, error) {
	re, err := regexp.Compile(vol.Spec.VgPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression %v for lvm volume %s: %v",
			vol.Spec.VgPattern, vol.Name, err)
	}
	capacity, err := strconv.Atoi(vol.Spec.Capacity)
	if err != nil {
		return nil, fmt.Errorf("invalid requested capacity %v for lvm volume %s: %v",
			vol.Spec.Capacity, vol.Name, err)
	}

	vgs, err := lvm.ListVolumeGroup(true)
	if err != nil {
		return nil, fmt.Errorf("failed to list vgs available on node: %v", err)
	}
	filteredVgs := make([]riov1.VolumeGroup, 0)
	for _, vg := range vgs {
		if !re.MatchString(vg.Name) {
			continue
		}
		// skip the vgs capacity comparison in case of thin provision enable volume
		if vol.Spec.ThinProvision != "yes" {
			// filter vgs having insufficient capacity.
			if vg.Free.Value() < int64(capacity) {
				continue
			}
		}
		filteredVgs = append(filteredVgs, vg)
	}

	// prioritize the volume group having less free space available.
	sort.SliceStable(filteredVgs, func(i, j int) bool {
		return filteredVgs[i].Free.Cmp(filteredVgs[j].Free) < 0
	})
	return filteredVgs, nil
}

func (r *VolumeReconciler) isDeletionCandidate(vol *riov1.Volume) bool {
	return vol.ObjectMeta.DeletionTimestamp != nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&riov1.Volume{}).
		Complete(r)
}
