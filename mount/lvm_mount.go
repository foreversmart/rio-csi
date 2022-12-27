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
	"errors"
	"fmt"
	"math"
	"os"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/mount/device/iolimit"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
	utilexec "k8s.io/utils/exec"
	"k8s.io/utils/mount"
)

// FormatAndMountVol formats and mounts the created volume to the desired mount path
func FormatAndMountVol(devicePath string, mountInfo *Info) error {
	mounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: utilexec.New()}

	err := mounter.FormatAndMount(devicePath, mountInfo.MountPath, mountInfo.FSType, mountInfo.MountOptions)
	if err != nil {
		klog.Errorf(
			"lvm: failed to mount volume %s [%s] to %s, error %v",
			devicePath, mountInfo.FSType, mountInfo.MountPath, err,
		)
		return err
	}

	return nil
}

// UmountVolume unmounts the volume and the corresponding mount path is removed
func UmountVolume(vol *apis.Volume, targetPath string) error {
	mounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: utilexec.New()}

	dev, ref, err := mount.GetDeviceNameFromMount(mounter, targetPath)
	if err != nil {
		klog.Errorf(
			"lvm: umount volume: failed to get device from mnt: %s\nError: %v",
			targetPath, err,
		)
		return err
	}

	// device has already been un-mounted, return successful
	if len(dev) == 0 || ref == 0 {
		klog.Warningf(
			"Warning: Unmount skipped because volume %s not mounted: %v",
			vol.Name, targetPath,
		)
		return nil
	}

	if pathExists, pathErr := mount.PathExists(targetPath); pathErr != nil {
		return fmt.Errorf("error checking if path exists: %v", pathErr)
	} else if !pathExists {
		klog.Warningf(
			"Warning: Unmount skipped because path does not exist: %v",
			targetPath,
		)
		return nil
	}

	if err = mounter.Unmount(targetPath); err != nil {
		klog.Errorf(
			"lvm: failed to unmount %s: path %s err: %v",
			vol.Name, targetPath, err,
		)
		return err
	}

	if err := os.RemoveAll(targetPath); err != nil {
		klog.Errorf("lvm: failed to remove mount path vol %s err : %v", vol.Name, err)
	}

	klog.Infof("umount done %s path %v", vol.Name, targetPath)

	return nil
}

func verifyMountRequest(vol *apis.Volume, mountpath string) (bool, error) {
	if len(mountpath) == 0 {
		return false, status.Error(codes.InvalidArgument, "verifyMount: mount path missing in request")
	}

	if vol.Finalizers == nil {
		return false, status.Error(codes.Internal, "verifyMount: volume is not ready to be mounted")
	}

	devicePath, err := GetVolumeDevPath(vol)
	if err != nil {
		klog.Errorf("can not get device for volume:%s dev %s err: %v",
			vol.Name, devicePath, err.Error())
		return false, status.Errorf(codes.Internal, "verifyMount: GetVolumePath failed %s", err.Error())
	}

	/*
	 * This check is the famous *Wall Of North*
	 * It will not let the volume to be mounted
	 * at more than two places. The volume should
	 * be unmounted before proceeding to the mount
	 * operation.
	 */
	currentMounts, err := GetMounts(devicePath)
	if err != nil {
		klog.Errorf("can not get mounts for volume:%s dev %s err: %v",
			vol.Name, devicePath, err.Error())
		return false, status.Errorf(codes.Internal, "verifyMount: Getmounts failed %s", err.Error())
	} else if len(currentMounts) >= 1 {
		// if device is already mounted at the mount point, return successful
		for _, mp := range currentMounts {
			if mp == mountpath {
				return true, nil
			}
		}

		// if it is not a shared volume, then it should not mounted to more than one path
		if vol.Spec.Shared != "yes" {
			klog.Errorf(
				"can not mount, volume:%s already mounted dev %s mounts: %v",
				vol.Name, devicePath, currentMounts,
			)
			return false, status.Errorf(codes.Internal, "verifyMount: device already mounted at %s", currentMounts)
		}
	}
	return false, nil
}

// MountVolume mounts the disk to the specified path
func MountVolume(vol *apis.Volume, mount *Info, podLVInfo *PodInfo) error {
	volume := vol.Spec.VolGroup + "/" + vol.Name
	mounted, err := verifyMountRequest(vol, mount.MountPath)
	if err != nil {
		return err
	}

	if mounted {
		klog.Infof("lvm : already mounted %s => %s", volume, mount.MountPath)
		return nil
	}

	devicePath := DevPath + volume

	err = FormatAndMountVol(devicePath, mount)
	if err != nil {
		return status.Errorf(
			codes.Internal,
			"failed to format and mount the volume error: %s",
			err.Error(),
		)
	}

	klog.Infof("lvm: volume %v mounted %v fs %v", volume, mount.MountPath, mount.FSType)

	if podLVInfo != nil {
		if err := setIOLimits(vol, podLVInfo, devicePath); err != nil {
			klog.Warningf("lvm: error setting io limits: podUid %s, device %s, err=%v", podLVInfo.UID, devicePath, err)
		} else {
			klog.Infof("lvm: io limits set for podUid %v, device %s", podLVInfo.UID, devicePath)
		}
	}

	return nil
}

// MountFilesystem mounts the disk to the specified path
func MountFilesystem(vol *apis.Volume, mount *Info, podinfo *PodInfo) error {
	if err := os.MkdirAll(mount.MountPath, 0755); err != nil {
		return status.Errorf(codes.Internal, "Could not create dir {%q}, err: %v", mount.MountPath, err)
	}

	return MountVolume(vol, mount, podinfo)
}

// MountBlock mounts the block disk to the specified path
func MountBlock(vol *apis.Volume, mountinfo *Info, podLVInfo *PodInfo) error {
	target := mountinfo.MountPath
	volume := vol.Spec.VolGroup + "/" + vol.Name
	devicePath := DevPath + volume

	mountopt := []string{"bind"}

	mounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: utilexec.New()}

	// Create the mount point as a file since bind mount device node requires it to be a file
	err := makeFile(target)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not create target file %q: %v", target, err)
	}

	// do the bind mount of the device at the target path
	if err := mounter.Mount(devicePath, target, "", mountopt); err != nil {
		if removeErr := os.RemoveAll(target); removeErr != nil {
			return status.Errorf(codes.Internal, "Could not remove mount target %q: %v", target, removeErr)
		}
		return status.Errorf(codes.Internal, "mount failed at %v err : %v", target, err)
	}

	klog.Infof("NodePublishVolume mounted block device %s at %s", devicePath, target)

	if podLVInfo != nil {
		if err := setIOLimits(vol, podLVInfo, devicePath); err != nil {
			klog.Warningf(": error setting io limits for podUid %s, device %s, err=%v", podLVInfo.UID, devicePath, err)
		} else {
			klog.Infof("lvm: io limits set for podUid %s, device %s", podLVInfo.UID, devicePath)
		}
	}
	return nil
}

func setIOLimits(vol *apis.Volume, podLVInfo *PodInfo, devicePath string) error {
	if podLVInfo == nil {
		return errors.New("PodInfo is missing. Skipping setting IOLimits")
	}
	capacityBytes, err := strconv.ParseUint(vol.Spec.Capacity, 10, 64)
	if err != nil {
		klog.Warning("error parsing Volume.Spec.Capacity. Skipping setting IOLimits", err)
		return err
	}
	capacityGB := uint64(math.Ceil(float64(capacityBytes) / (1024 * 1024 * 1024)))
	klog.Infof("Capacity of device in GB: %v", capacityGB)
	riops := getRIopsPerGB(vol.Spec.VolGroup) * capacityGB
	wiops := getWIopsPerGB(vol.Spec.VolGroup) * capacityGB
	rbps := getRBpsPerGB(vol.Spec.VolGroup) * capacityGB
	wbps := getWBpsPerGB(vol.Spec.VolGroup) * capacityGB
	klog.Infof("Setting iolimits for podUId %s, device %s: riops=%v, wiops=%v, rbps=%v, wbps=%v",
		podLVInfo.UID, devicePath, riops, wiops, rbps, wbps,
	)
	err = iolimit.SetIOLimits(&iolimit.Request{
		DeviceName:       devicePath,
		PodUid:           podLVInfo.UID,
		ContainerRuntime: getContainerRuntime(),
		IOLimit: &iolimit.IOMax{
			Riops: riops,
			Wiops: wiops,
			Rbps:  rbps,
			Wbps:  wbps,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func makeFile(pathname string) error {
	f, err := os.OpenFile(pathname, os.O_CREATE, os.FileMode(0644))
	defer func(f *os.File) {
		err = f.Close()
		klog.Errorf("failed to close file %s error: %v", f.Name(), err)
	}(f)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}
