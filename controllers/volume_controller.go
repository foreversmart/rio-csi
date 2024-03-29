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
	"path/filepath"
	riov1 "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/enums"
	"qiniu.io/rio-csi/lib/dd"
	"qiniu.io/rio-csi/lib/iscsi"
	"qiniu.io/rio-csi/lib/lvm"
	"qiniu.io/rio-csi/logger"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VolumeReconciler reconciles a Volume object
type VolumeReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	NodeID        string
	IscsiUsername string
	IscsiPassword string
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
	l.Info("reconcile volume" + req.Name + " " + req.Namespace)

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
		// logger.StdLog.Error(fmt.Errorf("nodeid: %s, vol node id %s is not same", r.NodeID, vol.Spec.OwnerNodeID), "vol is not on this node")
		return ctrl.Result{}, nil
	}

	err = r.syncVol(ctx, &vol)
	if err != nil {
		logger.StdLog.Error("sync vol error", err)
		// retry
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 30,
		}, nil
	}

	return ctrl.Result{}, nil
}

func (r *VolumeReconciler) syncVol(ctx context.Context, vol *riov1.Volume) error {
	l := log.FromContext(ctx)
	var err error
	// :remove
	// LVM Volume should be deleted. Check if deletion timestamp is set
	if r.isDeletionCandidate(vol) {
		return r.removeVolume(ctx, vol)
	}

	// if status is Pending then it means we are creating the volume.
	// Otherwise, we are just ignoring the event.
	switch vol.Status.State {
	case crd.StatusFailed:
		logger.StdLog.Error(nil, "Skipping retrying lvm volume provisioning as its already in failed state: %+v", vol.Status.Error)
		return nil
	case crd.StatusReady:
		l.Info("lvm volume already provisioned")
		return nil
	case crd.StatusCreated:
		err = r.cloneFromSource(ctx, vol)
		if err != nil {
			logger.StdLog.Error(err, "cloneFromSource", vol.Name)
			return err
		}

		return nil
	}

	err = r.createVolume(ctx, vol)
	if err != nil {
		logger.StdLog.Errorf("createVolume %s, error %v", vol.Name, err)
		return err
	}

	err = r.cloneFromSource(ctx, vol)
	if err != nil {
		logger.StdLog.Errorf("cloneFromSource %s, error %v", vol.Name, err)
		return err
	}

	// TODO retry check and turn into failed status
	if err != nil {
		// In case no vg available or lvm.CreateLVMVolume fails for all vgs, mark
		// the volume provisioning failed so that controller can reschedule it.
		vol.Status.Error = r.transformLVMError(err)
		return crd.UpdateVolInfoWithStatus(vol, crd.StatusFailed)
	}

	return nil
}

func (r *VolumeReconciler) removeVolume(ctx context.Context, vol *riov1.Volume) (err error) {
	// Unmount iscsi device
	lunStr := fmt.Sprintf("%d", vol.Spec.IscsiLun)
	_, err = iscsi.UnmountLun(vol.Spec.IscsiTarget, lunStr)
	if err != nil {
		logger.StdLog.Error(err)
		return err
	}

	// UnPublish volume
	_, err = iscsi.UnPublicBlockDevice(vol.Name)
	if err != nil {
		logger.StdLog.Error(err)
		return err
	}

	// remove disk iscsi target
	err = iscsi.DeleteTarget(vol.Spec.IscsiTarget)
	if err != nil {
		logger.StdLog.Error(err)
		return err
	}

	// remove lvm lv
	err = lvm.DeleteLVMVolume(vol)
	if err == nil {
		err = crd.RemoveVolFinalizer(vol)
	}
	return err
}

func (r *VolumeReconciler) createVolume(ctx context.Context, vol *riov1.Volume) (err error) {
	// if there is already a volGroup field set for lvmvolume resource,
	// we'll first try to create a volume in that volume group.
	if vol.Spec.VolGroup != "" {
		err = lvm.CreateLVMVolume(vol)
		if err != nil {
			logger.StdLog.Errorf("create lvm volume %s error %v", vol.Name, err)
		}
	}

	// create fails or VolGroup == empty
	if (vol.Spec.VolGroup != "" && err != nil) || vol.Spec.VolGroup == "" {
		vgs, vgErr := r.getVgPriorityList(vol)
		if vgErr != nil {
			logger.StdLog.Errorf("getVgPriorityList %s error %v", vol.Name, vgErr)
			return vgErr
		}

		if len(vgs) == 0 {
			err = fmt.Errorf("no vg available to serve volume request having regex=%q & capacity=%q",
				vol.Spec.VgPattern, vol.Spec.Capacity)
			logger.StdLog.Errorf("lvm volume %v - %v", vol.Name, err)
			return

		} else {
			for _, vg := range vgs {
				// first update volGroup field in lvm volume resource for ensuring
				// idempotency and avoiding volume leaks during crash.
				if vol, err = crd.UpdateVolGroup(vol, vg.Name); err != nil {
					logger.StdLog.Errorf("failed to update volGroup to %v: %v", vg.Name, err)
					return err
				}

				err = lvm.CreateLVMVolume(vol)
				if err != nil {
					// try next vgs to create volume
					logger.StdLog.Errorf("create lvm volume %s error %v", vol.Name, err)
					continue
				}

				// create lvm volume exist
				break
			}
		}
	}

	if vol.Spec.IscsiTarget == "" {
		// create volume target
		volumeTarget := iscsi.GenerateTargetName("volume", vol.Name)
		_, err = iscsi.CreateTarget(volumeTarget)
		if err != nil {
			logger.StdLog.Errorf("CreateTarget %s error %v", volumeTarget, err)
			return err
		}

		vol.Spec.IscsiTarget = volumeTarget
		vol, err = crd.UpdateVolume(vol)
		if err != nil {
			logger.StdLog.Error(err, fmt.Sprintf("UpdateVolume vol %s error:  %v",
				vol.Name, err))
			return err
		}
	}

	if err == nil && !vol.Spec.IscsiACLIsSet {
		// check ACL
		err = CreateTargetAcl(vol.Namespace, vol.Spec.IscsiTarget, r.IscsiUsername, r.IscsiPassword)
		if err != nil {
			logger.StdLog.Error(err, fmt.Sprintf("CreateTargetAcl %v", err))
			return err
		}

		vol.Spec.IscsiACLIsSet = true
		vol, err = crd.UpdateVolume(vol)
		if err != nil {
			logger.StdLog.Error(err, fmt.Sprintf("UpdateVolume vol %s error:  %v",
				vol.Name, err))
			return err
		}
	}

	// Iscsi publish volume
	if err == nil && vol.Spec.IscsiBlock == "" {
		device := getVolumeDevice(vol)
		// TODO check device exist
		_, err = iscsi.PublicBlockDevice(vol.Name, device)
		if err != nil {
			logger.StdLog.Error(err, fmt.Sprintf("PublicBlockDevice target %s, vol %s, device %s error: %v",
				vol.Spec.IscsiTarget, vol.Name, device, err))
			return err
		}

		// TODO block path
		// TODO update fail recover
		vol.Spec.IscsiBlock = vol.Name
		vol, err = crd.UpdateVolume(vol)
		if err != nil {
			logger.StdLog.Error(err, fmt.Sprintf("UpdateVolume vol %s error:  %v",
				vol.Name, err))
			return err
		}
	}

	// Iscsi Mount lun device
	if vol.Spec.IscsiLun == -1 {

		lunID := ""
		// TODO handle already exist error
		lunID, err = iscsi.MountLun(vol.Spec.IscsiTarget, vol.Name)
		if err != nil {
			logger.StdLog.Error(err, fmt.Sprintf("MountLun target %s, vol %s,  error: %v",
				vol.Spec.IscsiTarget, vol.Name, err))
			return err
		}

		lunIntID, parseErr := strconv.ParseInt(lunID, 10, 32)
		if parseErr != nil {
			logger.StdLog.Error(parseErr, fmt.Sprintf("MountLun ParseInt %s error: %v", lunID, parseErr))
			err = parseErr
			return
		}

		// TODO lun number
		vol.Spec.IscsiLun = int32(lunIntID)
		vol, err = crd.UpdateVolume(vol)
		if err != nil {
			logger.StdLog.Error(err, fmt.Sprintf("UpdateVolume vol %s error:  %v",
				vol.Name, err))
			return err
		}
	}

	err = crd.UpdateVolInfoWithStatus(vol, crd.StatusCreated)
	if err != nil {
		logger.StdLog.Error(err, "UpdateVolInfoWithStatus:", vol.Name)
		return err
	}

	return nil
}

func (r *VolumeReconciler) cloneFromSource(ctx context.Context, vol *riov1.Volume) (err error) {
	logger.StdLog.Infof("disk %s cloneFromSource", vol.Name)
	switch vol.Spec.DataSourceType {
	case enums.DataSourceTypeSnapshot:
		// empty data source skip to ready
		if vol.Spec.DataSource == "" {
			break
		}

		snapshotName := lvm.GetLVMSnapName(vol.Spec.DataSource)
		snapshotDevPath := lvm.GetDevPath(vol.Spec.VolGroup, snapshotName)
		volumeDevPath := lvm.GetVolumeDevPath(vol)
		// check path exist
		if exist, exErr := lvm.CheckPathExist(snapshotDevPath); !exist {
			logger.StdLog.Error(exErr, "snapshot dev path %s not exist", snapshotDevPath)
			return exErr
		}

		if exist, exErr := lvm.CheckPathExist(volumeDevPath); !exist {
			logger.StdLog.Error(exErr, "volume dev path %s not exist", snapshotDevPath)
			return exErr
		}

		logger.StdLog.Infof("disk dump %s to volume %s", snapshotDevPath, volumeDevPath)
		err = dd.DiskDump(snapshotDevPath, volumeDevPath)
		if err != nil {
			logger.StdLog.Error(err, "DiskDump error %s and %s", snapshotDevPath, volumeDevPath)
			return err
		}
		logger.StdLog.Infof("finish disk %s cloneFromSource", vol.Name)
	}

	err = crd.UpdateVolInfoWithStatus(vol, crd.StatusReady)
	if err != nil {
		logger.StdLog.Error(err, "UpdateVolInfoWithStatus:", vol.Name)
		return err
	}

	return nil
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

func getVolumeDevice(v *riov1.Volume) string {
	return filepath.Join("dev", v.Spec.VolGroup, v.Name)
}
