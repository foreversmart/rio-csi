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

package crd

import (
	"context"
	"github.com/pkg/errors"
	"os"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/lib/lvm/builder/volbuilder"
	"qiniu.io/rio-csi/logger"
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
	// RioFinalizer for the Volume CR
	RioFinalizer string = "rio.csi.io/finalizer"
	// VolGroupKey is key for LVM group name
	VolGroupKey string = "rio/lvm-group"
	// VolKey for the Snapshot CR to store Persistence Volume name
	VolKey string = "rio.csi.io/persistent-volume"
	// NodeKey will be used to insert Label in Volume CR
	NodeKey string = "kubernetes.io/nodename"
	// TopologyKey is supported topology key for the lvm driver
	TopologyKey string = "rio.csi.io/nodename"

	// StatusPending shows object has not handled yet
	StatusPending string = "Pending"
	// StatusCreated shows volume has finished created
	StatusCreated string = "Created"
	// StatusCloning shows volume is cloning data
	StatusCloning string = "Cloning"
	// StatusReady shows object has been processed
	StatusReady string = "Ready"
	// StatusFailed shows object operation has failed
	StatusFailed string = "Failed"
)

var (
	// RioNamespace is openebs system namespace
	RioNamespace string

	// NodeID is the NodeID of the node on which the pod is present
	NodeID string
)

func init() {

	RioNamespace = os.Getenv(RioNamespaceKey)
	if RioNamespace == "" {
		klog.Fatalf("NAMESPACE environment variable not set")
	}
	NodeID = os.Getenv("NODE_ID")
	if NodeID == "" {
		klog.Fatalf("NodeID environment variable not set")
	}

}

// ProvisionVolume creates a Volume CR,
// watcher for volume is present in CSI agent
func ProvisionVolume(vol *apis.Volume) (*apis.Volume, error) {
	options := metav1.CreateOptions{}
	result, err := client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).Create(context.Background(), vol, options)
	if err != nil {
		return nil, err
	}

	result.Status.State = StatusPending
	return UpdateVolumeStatus(result)
}

// UpdateVolumeStatus update volume status
func UpdateVolumeStatus(vol *apis.Volume) (*apis.Volume, error) {
	options := metav1.UpdateOptions{}
	res, err := client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).UpdateStatus(context.Background(), vol, options)
	if err == nil {
		klog.Infof("provisioned volume %s statue %s", vol.Name, vol.Status.State)
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

	err = client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).Delete(context.Background(), volumeID, options)
	if err == nil {
		klog.Infof("deprovisioned volume %s", volumeID)
	}

	return
}

// GetVolume fetches the given Volume
func GetVolume(volumeID string) (*apis.Volume, error) {
	getOptions := metav1.GetOptions{}
	vol, err := client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).Get(context.Background(), volumeID, getOptions)
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
		if vol.Status.State == StatusReady ||
			vol.Status.State == StatusFailed ||
			vol.Status.State == StatusCreated {
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
// the given volume. CreateLVMVolume request may call it again and
// again until volume is "Ready".
func GetVolumeState(volID string) (string, string, error) {
	vol, err := GetVolume(volID)

	if err != nil {
		return "", "", err
	}

	return vol.Spec.OwnerNodeID, vol.Status.State, nil
}

// UpdateVolInfoWithStatus updates Volume CR with node id and finalizer
func UpdateVolInfoWithStatus(vol *apis.Volume, state string) error {
	if vol.Finalizers != nil {
		return nil
	}

	var finalizers []string
	labels := map[string]string{NodeKey: NodeID}
	switch state {
	case StatusReady:
		finalizers = append(finalizers, RioFinalizer)
	}

	newVol, err := volbuilder.BuildFrom(vol).
		WithFinalizer(finalizers).
		WithLabels(labels).Build()

	if err != nil {
		return err
	}

	newVol, err = client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).Update(context.Background(), newVol, metav1.UpdateOptions{})
	if err != nil {
		logger.StdLog.Error(err)
		return err
	}

	newVol.Status.State = state
	_, err = client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).UpdateStatus(context.Background(), newVol, metav1.UpdateOptions{})

	return err
}

// UpdateVolume updates Volume
func UpdateVolume(vol *apis.Volume) (*apis.Volume, error) {
	return client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).Update(context.Background(), vol, metav1.UpdateOptions{})
}

// UpdateVolGroup updates Volume CR with volGroup name.
func UpdateVolGroup(vol *apis.Volume, vgName string) (*apis.Volume, error) {
	newVol, err := volbuilder.BuildFrom(vol).
		WithVolGroup(vgName).Build()
	if err != nil {
		return nil, err
	}

	return client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).Update(context.Background(), newVol, metav1.UpdateOptions{})
}

// RemoveVolFinalizer adds finalizer to Volume CR
func RemoveVolFinalizer(vol *apis.Volume) error {
	vol.Finalizers = nil

	_, err := client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).Update(context.Background(), vol, metav1.UpdateOptions{})
	return err
}

// ResizeVolume resizes the lvm volume
func ResizeVolume(vol *apis.Volume, newSize int64) error {

	vol.Spec.Capacity = strconv.FormatInt(int64(newSize), 10)

	_, err := client.DefaultClient.InternalClientSet.RioV1().Volumes(RioNamespace).Update(context.Background(), vol, metav1.UpdateOptions{})
	return err
}
