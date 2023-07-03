/*
Copyright © 2020 The OpenEBS Authors
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

package mount

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	utilexec "k8s.io/utils/exec"
	"k8s.io/utils/mount"
	"os"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/lib/mount/mtypes"
	"qiniu.io/rio-csi/logger"
	"sync"
)

var (
	Lock sync.Mutex
)

func MountVolume(vol *apis.Volume, info *mtypes.Info, iscsiUsername, iscsiPassword string) error {
	// TODO vol and pod on the same node to local mount
	connector, err := NewIscsiConnector(vol, iscsiUsername, iscsiPassword)
	if err != nil {
		return err
	}

	// add lock to limit iscsi connector
	Lock.Lock()
	defer Lock.Unlock()

	devicePath, rawDevicePaths, connectErr := connector.Connect()
	if connectErr != nil {
		logger.StdLog.Error(connectErr)
		return connectErr
	}

	if devicePath == "" {
		logger.StdLog.Error("connect reported success, but no path returned")
		return fmt.Errorf("connect reported success, but no path returned")
	}

	info.VolumeInfo.DevicePath = devicePath
	info.VolumeInfo.RawDevicePaths = rawDevicePaths

	// check mount node info has added or append to
	hasAddMountNode := false
	for _, mountNode := range vol.Spec.MountNodes {
		if mountNode.PodInfo.UID == info.PodInfo.UID {
			hasAddMountNode = true
		}
	}

	if !hasAddMountNode {
		vol.Spec.MountNodes = append(vol.Spec.MountNodes, info)
		vol, err := crd.UpdateVolume(vol)
		if err != nil {
			logger.StdLog.Errorf("update volume %s mount nodes error %v", vol.Name, err)
			return fmt.Errorf("update volume error %v", err)
		}
	}

	switch info.MountType {
	case mtypes.TypeBlock:
		// attempt block mount operation on the requested path
		err = MountBlock(vol, info.VolumeInfo, info.PodInfo)
	case mtypes.TypeFileSystem:
		// attempt filesystem mount operation on the requested path
		err = MountFilesystem(vol, info.VolumeInfo, info.PodInfo)
	}

	if err != nil {
		logger.StdLog.Errorf("node publish volume fails %v %v with error %v", info.PodInfo, info.VolumeInfo, err)
		return err
	}

	logger.StdLog.Info("node publish volume", info.PodInfo, info.VolumeInfo)
	return nil
}

// MountFilesystem mounts the disk to the specified path
func MountFilesystem(vol *apis.Volume, info *mtypes.VolumeInfo, podInfo *mtypes.PodInfo) error {
	target := info.MountPath
	devicePath := info.DevicePath

	// create dir as mount point
	if err := os.MkdirAll(target, 0755); err != nil {
		return status.Errorf(codes.Internal, "Could not create dir {%q}, err: %v", info.MountPath, err)
	}

	mounted, err := verifyMountRequest(vol, devicePath, info.MountPath)
	if err != nil {
		return err
	}

	if mounted {
		logger.StdLog.Infof("lvm : already mounted %s => %s", vol, info.MountPath)
		return nil
	}

	err = FormatAndMountVol(devicePath, info)
	if err != nil {
		return status.Errorf(
			codes.Internal,
			"failed to format and info the volume error: %s",
			err.Error(),
		)
	}

	logger.StdLog.Infof("lvm: volume %v mounted %v fs %v", vol, info.MountPath, info.FSType)

	if podInfo != nil {
		if err = setIOLimits(vol, podInfo, devicePath); err != nil {
			logger.StdLog.Warnf("lvm: error setting io limits: podUid %s, device %s, err=%v", podInfo.UID, devicePath, err)
		} else {
			logger.StdLog.Infof("lvm: io limits set for podUid %v, device %s", podInfo.UID, devicePath)
		}
	}

	return nil
}

// MountBlock mounts the block disk to the specified path
func MountBlock(vol *apis.Volume, info *mtypes.VolumeInfo, podLVInfo *mtypes.PodInfo) error {
	target := info.MountPath
	devicePath := info.DevicePath
	mountOpt := []string{"bind"}

	// Create the mount point as a file since bind mount device node requires it to be a file
	err := makeFile(target)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not create target file %q: %v", target, err)
	}

	mounted, err := verifyMountRequest(vol, devicePath, info.MountPath)
	if err != nil {
		return err
	}

	if mounted {
		logger.StdLog.Infof("lvm : already mounted %s => %s", vol, info.MountPath)
		return nil
	}

	mounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: utilexec.New()}
	// do the bind mount of the device at the target path
	if err := mounter.Mount(devicePath, target, "", mountOpt); err != nil {
		if removeErr := os.RemoveAll(target); removeErr != nil {
			return status.Errorf(codes.Internal, "Could not remove mount target %q: %v", target, removeErr)
		}
		return status.Errorf(codes.Internal, "mount failed at %v err : %v", target, err)
	}

	logger.StdLog.Infof("NodePublishVolume mounted block device %s at %s", devicePath, target)

	if podLVInfo != nil {
		if err = setIOLimits(vol, podLVInfo, devicePath); err != nil {
			logger.StdLog.Warnf(": error setting io limits for podUid %s, device %s, err=%v", podLVInfo.UID, devicePath, err)
		} else {
			logger.StdLog.Infof("lvm: io limits set for podUid %s, device %s", podLVInfo.UID, devicePath)
		}
	}
	return nil
}

// UmountVolume unmounts the volume and the corresponding mount path is removed
func UmountVolume(vol *apis.Volume, targetPath, iscsiUsername, iscsiPassword string, rawDevicePaths []string, isDisconnect bool) error {
	mounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: utilexec.New()}

	dev, ref, err := mount.GetDeviceNameFromMount(mounter, targetPath)
	if err != nil {
		logger.StdLog.Errorf(
			"lvm: umount volume: failed to get device from mnt: %s\nError: %v",
			targetPath, err,
		)
		return err
	}

	// device has already been un-mounted, return successful
	if len(dev) == 0 || ref == 0 {
		logger.StdLog.Warnf(
			"Warning: Unmount skipped because volume %s not mounted: %v",
			vol.Name, targetPath,
		)
		return nil
	}

	if pathExists, pathErr := mount.PathExists(targetPath); pathErr != nil {
		return fmt.Errorf("error checking if path exists: %v", pathErr)
	} else if !pathExists {
		logger.StdLog.Warnf(
			"Warning: Unmount skipped because path does not exist: %v",
			targetPath,
		)
		return nil
	}

	if err = mounter.Unmount(targetPath); err != nil {
		logger.StdLog.Errorf(
			"lvm: failed to unmount %s: path %s err: %v",
			vol.Name, targetPath, err,
		)
		return err
	}

	if err := os.RemoveAll(targetPath); err != nil {
		logger.StdLog.Errorf("lvm: failed to remove mount path vol %s err : %v", vol.Name, err)
	}

	ref--
	if ref != 0 || !isDisconnect {
		logger.StdLog.Infof("umount done  %s path %v", vol.Name, targetPath)
		return nil
	}

	connector, err := NewIscsiConnector(vol, iscsiUsername, iscsiPassword)
	if err != nil {
		return err
	}

	// disconnect volume device
	err = connector.DisconnectVolume(rawDevicePaths)
	if err != nil {

	}

	// disconnect iscsi session
	connector.Disconnect()

	logger.StdLog.Infof("umount done with disconnect iscsi %s path %v", vol.Name, targetPath)

	return nil
}
