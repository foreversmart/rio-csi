package lvm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog"
)

// LogicalVolume specifies attributes of a given lv that exists on the node.
type LogicalVolume struct {

	// Name of the lvm logical volume(name: pvc-213ca1e6-e271-4ec8-875c-c7def3a4908d)
	Name string

	// Full name of the lvm logical volume (fullName: linuxlvmvg/pvc-213ca1e6-e271-4ec8-875c-c7def3a4908d)
	FullName string

	// UUID denotes a unique identity of a lvm logical volume.
	UUID string

	// Size specifies the total size of logical volume in Bytes
	Size int64

	// Path specifies LVM logical volume path
	Path string

	// DMPath specifies device mapper path
	DMPath string

	// LVM logical volume device
	Device string

	// Name of the VG in which LVM logical volume is created
	VGName string

	// SegType specifies the type of Logical volume segment
	SegType string

	// Permission indicates the logical volume permission.
	// Permission has the following mapping between
	// int and string for its value:
	// [-1: "", 0: "unknown", 1: "writeable", 2: "read-only", 3: "read-only-override"]
	Permission int

	// BehaviourWhenFull indicates the behaviour of thin pools when it is full.
	// BehaviourWhenFull has the following mapping between int and string for its value:
	// [-1: "", 0: "error", 1: "queue"]
	BehaviourWhenFull int

	// HealthStatus indicates the health status of logical volumes.
	// HealthStatus has the following mapping between int and string for its value:
	// [0: "", 1: "partial", 2: "refresh needed", 3: "mismatches exist"]
	HealthStatus int

	// RaidSyncAction indicates the current synchronization action being performed for RAID
	// action.
	// RaidSyncAction has the following mapping between int and string for its value:
	// [-1: "", 0: "idle", 1: "frozen", 2: "resync", 3: "recover", 4: "check", 5: "repair"]
	RaidSyncAction int

	// ActiveStatus indicates the active state of logical volume
	ActiveStatus string

	// Host specifies the creation host of the logical volume, if known
	Host string

	// For thin volumes, the thin pool Logical volume for that volume
	PoolName string

	// UsedSizePercent specifies the percentage full for snapshot, cache
	// and thin pools and volumes if logical volume is active.
	UsedSizePercent float64

	// MetadataSize specifies the size of the logical volume that holds
	// the metadata for thin and cache pools.
	MetadataSize int64

	// MetadataUsedPercent specifies the percentage of metadata full if logical volume
	// is active for cache and thin pools.
	MetadataUsedPercent float64

	// SnapshotUsedPercent specifies the percentage full for snapshots  if
	// logical volume is active.
	SnapshotUsedPercent float64
}

// PhysicalVolume specifies attributes of a given pv that exists on the node.
type PhysicalVolume struct {
	// Name of the lvm physical volume.
	Name string

	// UUID denotes a unique identity of a lvm physical volume.
	UUID string

	// Size specifies the total size of physical volume in bytes
	Size resource.Quantity

	// DeviceSize specifies the size of underlying device in bytes
	DeviceSize resource.Quantity

	// MetadataSize specifies the size of smallest metadata area on this device in bytes
	MetadataSize resource.Quantity

	// MetadataFree specifies the free metadata area space on the device in bytes
	MetadataFree resource.Quantity

	// Free specifies the physical volume unallocated space in bytes
	Free resource.Quantity

	// Used specifies the physical volume allocated space in bytes
	Used resource.Quantity

	// Allocatable indicates whether the device can be used for allocation
	Allocatable string

	// Missing indicates whether the device is missing in the system
	Missing string

	// InUse indicates whether or not the physical volume is in use
	InUse string

	// Name of the volume group which uses this physical volume
	VGName string
}

// ExecError holds the process output along with underlying
// error returned by exec.CombinedOutput function.
type ExecError struct {
	Output []byte
	Err    error
}

// Error implements the error interface.
func (e *ExecError) Error() string {
	return fmt.Sprintf("%v - %v", string(e.Output), e.Err)
}

func newExecError(output []byte, err error) error {
	if err == nil {
		return nil
	}
	return &ExecError{
		Output: output,
		Err:    err,
	}
}

// CheckVolumeExists validates if lvm volume exists
func CheckVolumeExists(vol *apis.Volume) (bool, error) {
	devPath := GetVolumeDevMapperPath(vol)
	if _, err := os.Stat(devPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// builldVolumeResizeArgs returns resize command for the lvm volume
func buildVolumeResizeArgs(vol *apis.Volume, resizefs bool) []string {
	var LVMVolArg []string

	dev := GetVolumeDevPath(vol)
	size := vol.Spec.Capacity + "b"

	LVMVolArg = append(LVMVolArg, dev, "-L", size)

	if resizefs {
		LVMVolArg = append(LVMVolArg, "-r")
	}

	return LVMVolArg
}

// ResizeLVMVolume resizes the underlying LVM volume and FS if resizefs
// is set to true
// Note:
//  1. Triggering `lvextend <dev_path> -L <size> -r` multiple times with
//     same size will not return any errors
//  2. Triggering `lvextend <dev_path> -L <size>` more than one time will
//     cause errors
func ResizeLVMVolume(vol *apis.Volume, resizefs bool) error {

	// In case if resizefs is not enabled then check current size
	// before exapnding LVM volume(If volume is already expanded then
	// it might be error prone). This also makes ResizeVolume func
	// idempotent
	if !resizefs {
		desiredVolSize, err := strconv.ParseUint(vol.Spec.Capacity, 10, 64)
		if err != nil {
			return err
		}

		curVolSize, err := getLVSize(vol)
		if err != nil {
			return err
		}

		// Trigger resize only when desired volume size is greater than
		// current volume size else return
		if desiredVolSize <= curVolSize {
			return nil
		}
	}

	volume := vol.Spec.VolGroup + "/" + vol.Name

	args := buildVolumeResizeArgs(vol, resizefs)
	cmd := exec.Command(LVExtend, args...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		klog.Errorf(
			"lvm: could not resize the volume %v cmd %v error: %s", volume, args, string(out),
		)
	}

	return err
}

// getLVSize will return current LVM volume size in bytes
func getLVSize(vol *apis.Volume) (uint64, error) {
	VolumeName := vol.Spec.VolGroup + "/" + vol.Name

	args := []string{
		VolumeName,
		"--noheadings",
		"-o", "lv_size",
		"--units", "b",
		"--nosuffix",
	}

	cmd := exec.Command(LVList, args...)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return 0, errors.Wrapf(
			err,
			"could not get size of volume %v output: %s",
			VolumeName,
			string(raw),
		)
	}

	volSize, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, err
	}

	return volSize, nil
}

func decodeVgsJSON(raw []byte) ([]apis.VolumeGroup, error) {
	output := &struct {
		Report []struct {
			VolumeGroups []map[string]string `json:"vg"`
		} `json:"report"`
	}{}
	var err error
	if err = json.Unmarshal(raw, output); err != nil {
		return nil, err
	}

	if len(output.Report) != 1 {
		return nil, fmt.Errorf("expected exactly one lvm report")
	}

	items := output.Report[0].VolumeGroups
	vgs := make([]apis.VolumeGroup, 0, len(items))
	for _, item := range items {
		var vg apis.VolumeGroup
		if vg, err = parseVolumeGroup(item); err != nil {
			return vgs, err
		}
		vgs = append(vgs, vg)
	}
	return vgs, nil
}

func parseVolumeGroup(m map[string]string) (apis.VolumeGroup, error) {
	var vg apis.VolumeGroup
	var count int
	var sizeBytes int64
	var err error

	vg.Name = m[VGName]
	vg.UUID = m[VGUUID]

	int32Map := map[string]*int32{
		VGPVvount:           &vg.PVCount,
		VGLvCount:           &vg.LVCount,
		VGMaxLv:             &vg.MaxLV,
		VGMaxPv:             &vg.MaxPV,
		VGSnapCount:         &vg.SnapCount,
		VGMissingPvCount:    &vg.MissingPVCount,
		VGMetadataCount:     &vg.MetadataCount,
		VGMetadataUsedCount: &vg.MetadataUsedCount,
	}
	for key, value := range int32Map {
		count, err = strconv.Atoi(m[key])
		if err != nil {
			err = fmt.Errorf("invalid format of %v=%v for vg %v: %v", key, m[key], vg.Name, err)
		}
		*value = int32(count)
	}

	resQuantityMap := map[string]*resource.Quantity{
		VGSize:             &vg.Size,
		VGFreeSize:         &vg.Free,
		VGMetadataSize:     &vg.MetadataSize,
		VGMetadataFreeSize: &vg.MetadataFree,
	}

	for key, value := range resQuantityMap {
		sizeBytes, err = strconv.ParseInt(
			strings.TrimSuffix(strings.ToLower(m[key]), "b"),
			10, 64)
		if err != nil {
			err = fmt.Errorf("invalid format of %v=%v for vg %v: %v", key, m[key], vg.Name, err)
		}
		quantity := resource.NewQuantity(sizeBytes, resource.BinarySI)
		*value = *quantity //
	}

	vg.Permission = getIntFieldValue(VGPermissions, m[VGPermissions])
	vg.AllocationPolicy = getIntFieldValue(VGAllocationPolicy, m[VGAllocationPolicy])

	return vg, err
}

// This function returns the integer equivalent for different string values for the LVM component(vg,lv) field.
// -1 represents undefined.
func getIntFieldValue(fieldName, fieldValue string) int {
	mv := -1
	for i, v := range Enums[fieldName] {
		if v == fieldValue {
			mv = i
			break
		}
	}
	return mv
}

// ReloadLVMMetadataCache refreshes lvmetad daemon cache used for
// serving vgs or other lvm utility.
func ReloadLVMMetadataCache() error {
	args := []string{"--cache"}
	cmd := exec.Command(PVScan, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("lvm: reload lvm metadata cache: %v - %v", string(output), err)
		return err
	}
	return nil
}

// ListVolumeGroup invokes `vgs` to list all the available volume
// groups in the node.
//
// In case reloadCache is false, we skip refreshing lvm metadata cache.
func ListVolumeGroup(reloadCache bool) ([]apis.VolumeGroup, error) {
	if reloadCache {
		if err := ReloadLVMMetadataCache(); err != nil {
			return nil, err
		}
	}

	args := []string{
		"--options", "vg_all",
		"--reportformat", "json",
		"--units", "b",
	}
	cmd := exec.Command(VGList, args...)
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	b := &bytes.Buffer{}
	// remove device not exist error
	for _, line := range lines {
		if strings.Contains(line, "No such device or address") {
			continue
		}
		b.WriteString(line)
	}

	if err != nil {
		klog.Errorf("lvm: list volume group cmd %v: %v", args, err)
		return nil, err
	}
	return decodeVgsJSON(b.Bytes())
}

// Function to get LVM Logical volume device
// It returns LVM logical volume device(dm-*).
// This is used as a label in metrics(lvm_lv_total_size) which helps us to map lv_name to device.
//
// Example: pvc-f147582c-adbd-4015-8ca9-fe3e0a4c2452(lv_name) -> dm-0(device)
func getLvDeviceName(path string) (string, error) {
	dmPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		klog.Errorf("failed to resolve device mapper from lv path %v: %v", path, err)
		return "", err
	}
	deviceName := strings.Split(dmPath, "/")
	return deviceName[len(deviceName)-1], nil
}

// To parse the output of lvs command and store it in LogicalVolume
// It returns LogicalVolume.
//
//	Example: LogicalVolume{
//			Name:               "pvc-082c7975-9af2-4a50-9d24-762612b35f94",
//			FullName:           "vg_thin/pvc-082c7975-9af2-4a50-9d24-762612b35f94"
//			UUID:               "FBqcEe-Ln72-SmWO-fR4j-t4Ga-1Y90-0vieKW"
//			Size:                4294967296,
//			Path:                "/dev/vg_thin/pvc-082c7975-9af2-4a50-9d24-762612b35f94",
//			DMPath:              "/dev/mapper/vg_thin-pvc--082c7975--9af2--4a50--9d24--762612b35f94"
//			Device:              "dm-5"
//			VGName:              "vg_thin"
//			SegType:             "thin"
//			Permission:          1
//			BehaviourWhenFull:   -1
//			HealthStatus:        0
//			RaidSyncAction:      -1
//			ActiveStatus:        "active"
//			Host:                "node1-virtual-machine"
//			PoolName:            "vg_thin_thinpool"
//			UsedSizePercent:     0
//			MetadataSize:        0
//			MetadataUsedPercent: 0
//			SnapshotUsedPercent: 0
//		}
func parseLogicalVolume(m map[string]string) (LogicalVolume, error) {
	var lv LogicalVolume
	var err error
	var sizeBytes int64
	var count float64

	lv.Name = m[LVName]
	lv.FullName = m[LVFullName]
	lv.UUID = m[LVUUID]
	lv.Path = m[LVPath]
	lv.DMPath = m[LVDmPath]
	lv.VGName = m[VGName]
	lv.ActiveStatus = m[LVActive]

	int64Map := map[string]*int64{
		LVSize:         &lv.Size,
		LVMetadataSize: &lv.MetadataSize,
	}
	for key, value := range int64Map {
		// Check if the current LV is not a thin pool. If not then
		// metadata size will not be present as metadata is only
		// stored for thin pools.
		if m[LVSegtype] != LVThinPool && key == LVMetadataSize {
			sizeBytes = 0
		} else {
			sizeBytes, err = strconv.ParseInt(strings.TrimSuffix(strings.ToLower(m[key]), "b"), 10, 64)
			if err != nil {
				err = fmt.Errorf("invalid format of %v=%v for vg %v: %v", key, m[key], lv.Name, err)
				return lv, err
			}
		}
		*value = sizeBytes
	}

	lv.SegType = m[LVSegtype]
	lv.Host = m[LVHost]
	lv.PoolName = m[LVPool]
	lv.Permission = getIntFieldValue(LVPermissions, m[LVPermissions])
	lv.BehaviourWhenFull = getIntFieldValue(LVWhenFull, m[LVWhenFull])
	lv.HealthStatus = getIntFieldValue(LVHealthStatus, m[LVHealthStatus])
	lv.RaidSyncAction = getIntFieldValue(RaidSyncAction, m[RaidSyncAction])

	float64Map := map[string]*float64{
		LVDataPercent:     &lv.UsedSizePercent,
		LVMetadataPercent: &lv.MetadataUsedPercent,
		LVSnapPercent:     &lv.SnapshotUsedPercent,
	}
	for key, value := range float64Map {
		if m[key] == "" {
			count = 0
		} else {
			count, err = strconv.ParseFloat(m[key], 64)
			if err != nil {
				err = fmt.Errorf("invalid format of %v=%v for lv %v: %v", key, m[key], lv.Name, err)
				return lv, err
			}
		}
		*value = count
	}

	return lv, err
}

// decodeLvsJSON([]bytes): Decode json format and pass the unmarshalled json to parseLogicalVolume to store logical volumes in LogicalVolume
//
// Output of lvs command will be in json format:
//
//	{
//		"report": [
//			{
//				"lv": [
//						{
//							"lv_name":"pvc-082c7975-9af2-4a50-9d24-762612b35f94",
//							...
//						}
//					]
//			}
//		]
//	}
//
// This function is used to decode the output of lvs command.
// It returns []LogicalVolume.
//
//	Example: []LogicalVolume{
//		{
//			Name:               "pvc-082c7975-9af2-4a50-9d24-762612b35f94",
//			FullName:           "vg_thin/pvc-082c7975-9af2-4a50-9d24-762612b35f94"
//			UUID:               "FBqcEe-Ln72-SmWO-fR4j-t4Ga-1Y90-0vieKW"
//			Size:                4294967296,
//			Path:                "/dev/vg_thin/pvc-082c7975-9af2-4a50-9d24-762612b35f94",
//			DMPath:              "/dev/mapper/vg_thin-pvc--082c7975--9af2--4a50--9d24--762612b35f94"
//			Device:              "dm-5"
//			VGName:              "vg_thin"
//			SegType:             "thin"
//			Permission:          1
//			BehaviourWhenFull:   -1
//			HealthStatus:        0
//			RaidSyncAction:      -1
//			ActiveStatus:        "active"
//			Host:                "node1-virtual-machine"
//			PoolName:            "vg_thin_thinpool"
//			UsedSizePercent:     0
//			MetadataSize:        0
//			MetadataUsedPercent: 0
//			SnapshotUsedPercent: 0
//		}
//	}
func decodeLvsJSON(raw []byte) ([]LogicalVolume, error) {
	output := &struct {
		Report []struct {
			LogicalVolumes []map[string]string `json:"lv"`
		} `json:"report"`
	}{}
	var err error
	if err = json.Unmarshal(raw, output); err != nil {
		return nil, err
	}

	if len(output.Report) != 1 {
		return nil, fmt.Errorf("expected exactly one lvm report")
	}

	items := output.Report[0].LogicalVolumes
	lvs := make([]LogicalVolume, 0, len(items))
	for _, item := range items {
		var lv LogicalVolume
		if lv, err = parseLogicalVolume(item); err != nil {
			return lvs, err
		}
		deviceName, err := getLvDeviceName(lv.Path)
		if err != nil {
			klog.Error(err)
			return nil, err
		}
		lv.Device = deviceName
		lvs = append(lvs, lv)
	}
	return lvs, nil
}

func ListLVMLogicalVolume() ([]LogicalVolume, error) {
	args := []string{
		"--options", "lv_all,vg_name,segtype",
		"--reportformat", "json",
		"--units", "b",
	}
	cmd := exec.Command(LVList, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("lvm: error while running command %s %v: %v", LVList, args, err)
		return nil, err
	}
	return decodeLvsJSON(output)
}

/*
ListLVMPhysicalVolume invokes `pvs` to list all the available LVM physical volumes in the node.
*/
func ListLVMPhysicalVolume() ([]PhysicalVolume, error) {
	if err := ReloadLVMMetadataCache(); err != nil {
		return nil, err
	}

	args := []string{
		"--options", "pv_all,vg_name",
		"--reportformat", "json",
		"--units", "b",
	}
	cmd := exec.Command(PVList, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("lvm: error while running command %s %v: %v", PVList, args, err)
		return nil, err
	}
	return decodePvsJSON(output)
}

// To parse the output of pvs command and store it in PhysicalVolume
// It returns PhysicalVolume.
//
//	Example: PhysicalVolume{
//			Name:         "/dev/sdc",
//	     UUID:         "UAdQl0-dK00-gM1V-6Vda-zYeu-XUdQ-izs8KW"
//			Size:         21441282048
//			Used:         8657043456
//			Free:         12784238592
//			MetadataSize: 1044480
//			MetadataFree: 518656
//			DeviceSize:   21474836480
//			Allocatable:  "allocatable"
//			InUse:        "used"
//			Missing:      ""
//			VGName:       "vg_thin"
//		}
func parsePhysicalVolume(m map[string]string) (PhysicalVolume, error) {
	var pv PhysicalVolume
	var err error
	var sizeBytes int64

	pv.Name = m[PVName]
	pv.UUID = m[PVUUID]
	pv.InUse = m[PVInUse]
	pv.Allocatable = m[PVAllocatable]
	pv.Missing = m[PVMissing]
	pv.VGName = m[VGName]

	resQuantityMap := map[string]*resource.Quantity{
		PVSize:             &pv.Size,
		PVFreeSize:         &pv.Free,
		PVUsedSize:         &pv.Used,
		PVMetadataSize:     &pv.MetadataSize,
		PVMetadataFreeSize: &pv.MetadataFree,
		PVDeviceSize:       &pv.DeviceSize,
	}

	for key, value := range resQuantityMap {
		sizeBytes, err = strconv.ParseInt(
			strings.TrimSuffix(strings.ToLower(m[key]), "b"),
			10, 64)
		if err != nil {
			err = fmt.Errorf("invalid format of %v=%v for pv %v: %v", key, m[key], pv.Name, err)
			return pv, err
		}
		quantity := resource.NewQuantity(sizeBytes, resource.BinarySI)
		*value = *quantity
	}

	return pv, err
}

// decodeLvsJSON([]bytes): Decode json format and pass the unmarshalled json to parsePhysicalVolume to store physical volumes in PhysicalVolume
//
// Output of pvs command will be in json format:
//
//	{
//		"report": [
//			{
//				"pv": [
//						{
//							"pv_name":"/dev/sdc",
//							...
//						}
//					]
//			}
//		]
//	}
//
// This function is used to decode the output of pvs command.
// It returns []PhysicalVolume.
//
//	Example: []PhysicalVolume{
//		{
//			Name:         "/dev/sdc",
//	     UUID:         "UAdQl0-dK00-gM1V-6Vda-zYeu-XUdQ-izs8KW"
//			Size:         21441282048
//			Used:         8657043456
//			Free:         12784238592
//			MetadataSize: 1044480
//			MetadataFree: 518656
//			DeviceSize:   21474836480
//			Allocatable:  "allocatable"
//			InUse:        "used"
//			Missing:      ""
//			VGName:       "vg_thin"
//		}
//	}
func decodePvsJSON(raw []byte) ([]PhysicalVolume, error) {
	output := &struct {
		Report []struct {
			PhysicalVolume []map[string]string `json:"pv"`
		} `json:"report"`
	}{}
	var err error
	if err = json.Unmarshal(raw, output); err != nil {
		return nil, err
	}

	if len(output.Report) != 1 {
		return nil, fmt.Errorf("expected exactly one lvm report")
	}

	items := output.Report[0].PhysicalVolume
	pvs := make([]PhysicalVolume, 0, len(items))
	for _, item := range items {
		var pv PhysicalVolume
		if pv, err = parsePhysicalVolume(item); err != nil {
			return pvs, err
		}
		pvs = append(pvs, pv)
	}
	return pvs, nil
}

// lvThinExists verifies if thin pool/volume already exists for given volumegroup
func lvThinExists(vg string, name string) bool {
	cmd := exec.Command("lvs", vg+"/"+name, "--noheadings", "-o", "lv_name")
	out, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("failed to list existing volumes:%v", err)
		return false
	}
	return name == strings.TrimSpace(string(out))
}

// snapshotExists checks if a snapshot volume exists for the given volumegroup
// and snapshot name.
func isSnapshotExists(vg, snapVolumeName string) (bool, error) {
	cmd := exec.Command("lvs", vg+"/"+snapVolumeName, "--noheadings", "-o", "lv_name")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	return snapVolumeName == strings.TrimSpace(string(out)), nil
}

// getVGSize get the size in bytes for given volumegroup name
func getVGSize(vgname string) string {
	cmd := exec.Command("vgs", vgname, "--noheadings", "-o", "vg_free", "--units", "b", "--nosuffix")
	out, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("failed to list existing volumegroup:%v , %v", vgname, err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

// getThinPoolSize gets size for a given volumegroup, compares it with
// the requested volume size and returns the minimum size as a thin pool size
func getThinPoolSize(vgname, volsize string) string {
	outStr := getVGSize(vgname)
	vgFreeSize, err := strconv.ParseInt(strings.TrimSpace(string(outStr)), 10, 64)
	if err != nil {
		klog.Errorf("failed to convert vg_size to int, got size,:%v , %v", outStr, err)
		return ""
	}

	volSize, err := strconv.ParseInt(strings.TrimSpace(string(volsize)), 10, 64)
	if err != nil {
		klog.Errorf("failed to convert volsize to int, got size,:%v , %v", volSize, err)
		return ""
	}

	if vgFreeSize < volSize {
		// reducing 268435456 bytes (256Mi) from the total byte size to round off
		// blocks extent
		return fmt.Sprint(vgFreeSize-MinExtentRoundOffSize) + "b"
	}
	return volsize + "b"
}

// removeVolumeFilesystem will erases the filesystem signature from lvm volume
func removeVolumeFilesystem(Volume *apis.Volume) error {
	devicePath := filepath.Join(DevPath, Volume.Spec.VolGroup, Volume.Name)

	// wipefs erases the filesystem signature from the lvm volume
	// -a    wipe all magic strings
	// -f    force erasure
	// Command: wipefs -af /dev/lvmvg/volume1
	cleanCommand := exec.Command(BlockCleanerCommand, "-af", devicePath)
	output, err := cleanCommand.CombinedOutput()
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to wipe filesystem on device path: %s resp: %s",
			devicePath,
			string(output),
		)
	}
	klog.V(4).Infof("Successfully wiped filesystem on device path: %s", devicePath)
	return nil
}
