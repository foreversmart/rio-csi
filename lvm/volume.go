// Copyright © 2019 The OpenEBS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lvm

import (
	"context"
	"github.com/pkg/errors"
	"os"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/lvm/builder/volbuilder"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	// RioNamespaceKey is the environment variable to get openebs namespace
	//
	// This environment variable is set via kubernetes downward API
	RioNamespaceKey string = "RIO_CSI_NAMESPACE"
	// GoogleAnalyticsKey This environment variable is set via env
	GoogleAnalyticsKey string = "OPENEBS_IO_ENABLE_ANALYTICS"
	// LVMFinalizer for the Volume CR
	LVMFinalizer string = "lvm.openebs.io/finalizer"
	// VolGroupKey is key for LVM group name
	VolGroupKey string = "openebs.io/volgroup"
	// LVMVolKey for the LVMSnapshot CR to store Persistence Volume name
	LVMVolKey string = "openebs.io/persistent-volume"
	// LVMNodeKey will be used to insert Label in Volume CR
	LVMNodeKey string = "kubernetes.io/nodename"
	// LVMTopologyKey is supported topology key for the lvm driver
	LVMTopologyKey string = "openebs.io/nodename"
	// LVMStatusPending shows object has not handled yet
	LVMStatusPending string = "Pending"
	// LVMStatusFailed shows object operation has failed
	LVMStatusFailed string = "Failed"
	// LVMStatusReady shows object has been processed
	LVMStatusReady string = "Ready"
	// OpenEBSCasTypeKey for the cas-type label
	OpenEBSCasTypeKey string = "openebs.io/cas-type"
	// LVMCasTypeName for the name of the cas-type
	LVMCasTypeName string = "localpv-lvm"
)

var (
	// RioNamespace is openebs system namespace
	RioNamespace string

	// NodeID is the NodeID of the node on which the pod is present
	NodeID string

	// GoogleAnalyticsEnabled should send google analytics or not
	GoogleAnalyticsEnabled string
)

func init() {

	RioNamespace = os.Getenv(RioNamespaceKey)
	if RioNamespace == "" {
		klog.Fatalf("LVM_NAMESPACE environment variable not set")
	}
	NodeID = os.Getenv("NODE_ID")
	if NodeID == "" {
		klog.Fatalf("NodeID environment variable not set")
	}

	GoogleAnalyticsEnabled = os.Getenv(GoogleAnalyticsKey)
}

// ProvisionVolume creates a Volume CR,
// watcher for volume is present in CSI agent
func ProvisionVolume(vol *apis.Volume) (*apis.Volume, error) {
	options := metav1.CreateOptions{}
	res, err := client.DefaultClient.ClientSet.RioV1().Volumes(RioNamespace).Create(context.Background(), vol, options)
	if err == nil {
		klog.Infof("provisioned volume %s", vol.Name)
	}

	return res, err
}

// DeleteVolume deletes the corresponding LVM Volume CR
func DeleteVolume(volumeID string) (err error) {
	if volumeID == "" {
		return errors.New(
			"failed to delete csivolume: missing vol name",
		)
	}

	deletePropagation := metav1.DeletePropagationForeground
	options := metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	}

	err = client.DefaultClient.ClientSet.RioV1().Volumes(RioNamespace).Delete(context.Background(), volumeID, options)
	if err == nil {
		klog.Infof("deprovisioned volume %s", volumeID)
	}

	return
}

// GetVolume fetches the given Volume
func GetVolume(volumeID string) (*apis.Volume, error) {
	getOptions := metav1.GetOptions{}
	vol, err := client.DefaultClient.ClientSet.RioV1().Volumes(RioNamespace).Get(context.Background(), volumeID, getOptions)
	return vol, err
}

// WaitForVolumeProcessed waits till the lvm volume becomes
// ready or failed (i.e reaches to terminal state).
func WaitForVolumeProcessed(ctx context.Context, volumeID string) (*apis.Volume, error) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, status.FromContextError(ctx.Err()).Err()
		case <-timer.C:
		}
		vol, err := GetVolume(volumeID)
		if err != nil {
			return nil, status.Errorf(codes.Aborted,
				"lvm: wait failed, not able to get the volume %s %s", volumeID, err.Error())
		}
		if vol.Status.State == LVMStatusReady ||
			vol.Status.State == LVMStatusFailed {
			return vol, nil
		}
		timer.Reset(1 * time.Second)
	}
}

// WaitForVolumeDestroy waits till the lvm volume gets deleted.
func WaitForVolumeDestroy(ctx context.Context, volumeID string) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
		case <-timer.C:
		}
		_, err := GetVolume(volumeID)
		if err != nil {
			if k8serror.IsNotFound(err) {
				return nil
			}
			return status.Errorf(codes.Aborted,
				"lvm: destroy wait failed, not able to get the volume %s %s", volumeID, err.Error())
		}
		timer.Reset(1 * time.Second)
	}
}

// GetVolumeState returns Volume OwnerNode and State for
// the given volume. CreateVolume request may call it again and
// again until volume is "Ready".
func GetVolumeState(volID string) (string, string, error) {
	vol, err := GetVolume(volID)

	if err != nil {
		return "", "", err
	}

	return vol.Spec.OwnerNodeID, vol.Status.State, nil
}

// UpdateVolInfo updates Volume CR with node id and finalizer
func UpdateVolInfo(vol *apis.Volume, state string) error {
	if vol.Finalizers != nil {
		return nil
	}

	var finalizers []string
	labels := map[string]string{LVMNodeKey: NodeID}
	switch state {
	case LVMStatusReady:
		finalizers = append(finalizers, LVMFinalizer)
	}
	newVol, err := volbuilder.BuildFrom(vol).
		WithFinalizer(finalizers).
		WithVolumeStatus(state).
		WithLabels(labels).Build()

	if err != nil {
		return err
	}

	_, err = client.DefaultClient.ClientSet.RioV1().Volumes(RioNamespace).Update(context.Background(), newVol, metav1.UpdateOptions{})

	return err
}

// UpdateVolGroup updates Volume CR with volGroup name.
func UpdateVolGroup(vol *apis.Volume, vgName string) (*apis.Volume, error) {
	newVol, err := volbuilder.BuildFrom(vol).
		WithVolGroup(vgName).Build()
	if err != nil {
		return nil, err
	}

	return client.DefaultClient.ClientSet.RioV1().Volumes(RioNamespace).Update(context.Background(), newVol, metav1.UpdateOptions{})
}

// RemoveVolFinalizer adds finalizer to Volume CR
func RemoveVolFinalizer(vol *apis.Volume) error {
	vol.Finalizers = nil

	_, err := client.DefaultClient.ClientSet.RioV1().Volumes(RioNamespace).Update(context.Background(), vol, metav1.UpdateOptions{})
	return err
}

// ResizeVolume resizes the lvm volume
func ResizeVolume(vol *apis.Volume, newSize int64) error {

	vol.Spec.Capacity = strconv.FormatInt(int64(newSize), 10)

	_, err := client.DefaultClient.ClientSet.RioV1().Volumes(RioNamespace).Update(context.Background(), vol, metav1.UpdateOptions{})
	return err
}

// ProvisionSnapshot creates a LVMSnapshot CR
//func ProvisionSnapshot(snap *apis.LVMSnapshot) error {
//	_, err := snapbuilder.NewKubeclient().WithNamespace(RioNamespace).Create(snap)
//	if err == nil {
//		klog.Infof("provosioned snapshot %s", snap.Name)
//	}
//	return err
//}

// DeleteSnapshot deletes the LVMSnapshot CR
//func DeleteSnapshot(snapName string) error {
//	err := snapbuilder.NewKubeclient().WithNamespace(RioNamespace).Delete(snapName)
//	if err == nil {
//		klog.Infof("deprovisioned snapshot %s", snapName)
//	}
//
//	return err
//}

//// GetLVMSnapshot fetches the given LVM snapshot
//func GetLVMSnapshot(snapID string) (*apis.LVMSnapshot, error) {
//	getOptions := metav1.GetOptions{}
//	snap, err := snapbuilder.NewKubeclient().WithNamespace(RioNamespace).Get(snapID, getOptions)
//	return snap, err
//}
//
//// GetSnapshotForVolume fetches all the snapshots for the given volume
//func GetSnapshotForVolume(volumeID string) (*apis.LVMSnapshotList, error) {
//	listOptions := metav1.ListOptions{
//		LabelSelector: LVMVolKey + "=" + volumeID,
//	}
//	snapList, err := snapbuilder.NewKubeclient().WithNamespace(RioNamespace).List(listOptions)
//	return snapList, err
//}
//
//// GetLVMSnapshotStatus returns the status of LVMSnapshot
//func GetLVMSnapshotStatus(snapID string) (string, error) {
//	getOptions := metav1.GetOptions{}
//	snap, err := snapbuilder.NewKubeclient().WithNamespace(RioNamespace).Get(snapID, getOptions)
//	if err != nil {
//		klog.Errorf("Get snapshot failed %s err: %s", snap.Name, err.Error())
//		return "", err
//	}
//	return snap.Status.State, nil
//}
//
//// UpdateSnapInfo updates LVMSnapshot CR with node id and finalizer
//func UpdateSnapInfo(snap *apis.LVMSnapshot) error {
//	finalizers := []string{LVMFinalizer}
//	labels := map[string]string{
//		LVMNodeKey: NodeID,
//	}
//
//	if snap.Finalizers != nil {
//		return nil
//	}
//
//	newSnap, err := snapbuilder.BuildFrom(snap).
//		WithFinalizer(finalizers).
//		WithLabels(labels).Build()
//
//	newSnap.Status.State = LVMStatusReady
//
//	if err != nil {
//		klog.Errorf("Update snapshot failed %s err: %s", snap.Name, err.Error())
//		return err
//	}
//
//	_, err = snapbuilder.NewKubeclient().WithNamespace(RioNamespace).Update(newSnap)
//	return err
//}
//
//// RemoveSnapFinalizer adds finalizer to LVMSnapshot CR
//func RemoveSnapFinalizer(snap *apis.LVMSnapshot) error {
//	snap.Finalizers = nil
//
//	_, err := snapbuilder.NewKubeclient().WithNamespace(RioNamespace).Update(snap)
//	return err
//}
