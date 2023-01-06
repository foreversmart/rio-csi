package lvm

import (
	"os/exec"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/logger"
	"strings"
)

func buildLVMSnapCreateArgs(snap *apis.Snapshot) []string {
	var LVMSnapArg []string

	volName := snap.Labels[crd.VolKey]
	volPath := DevPath + snap.Spec.VolGroup + "/" + volName
	size := snap.Spec.SnapSize + "b"

	LVMSnapArg = append(LVMSnapArg,
		// snapshot argument
		"--snapshot",
		// name of snapshot
		"--name", getLVMSnapName(snap.Name),
		// set the permission to make the snapshot read-only. By default LVM snapshots are RW
		"--permission", "r",
		// volume to snapshot
		volPath,
	)

	// When creating a thin snapshot volume, you do not specify the size of the volume.
	// If you specify a size parameter, the snapshot that will be created will not
	// be a thin snapshot volume and will not use the thin pool for storing data.
	if len(snap.Spec.SnapSize) != 0 {
		// size of the snapshot, will be same or less than source volume
		LVMSnapArg = append(LVMSnapArg, "--size", size)
	}
	return LVMSnapArg
}

func buildLVMSnapDestroyArgs(snap *apis.Snapshot) []string {
	var LVMSnapArg []string

	dev := DevPath + snap.Spec.VolGroup + "/" + getLVMSnapName(snap.Name)

	LVMSnapArg = append(LVMSnapArg, "-y", dev)

	return LVMSnapArg
}

// CreateSnapshot creates the lvm volume snapshot
func CreateSnapshot(snap *apis.Snapshot) error {

	volume := snap.Labels[crd.VolKey]

	snapVolume := snap.Spec.VolGroup + "/" + getLVMSnapName(snap.Name)

	args := buildLVMSnapCreateArgs(snap)
	cmd := exec.Command(LVCreate, args...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		logger.StdLog.Errorf("lvm: could not create snapshot %s cmd %v error: %s", snapVolume, args, string(out))
		return err
	}

	logger.StdLog.Infof("created snapshot %s from %s", snapVolume, volume)
	return nil

}

// DestroySnapshot deletes the lvm volume snapshot
func DestroySnapshot(snap *apis.Snapshot) error {
	snapVolume := snap.Spec.VolGroup + "/" + getLVMSnapName(snap.Name)

	ok, err := isSnapshotExists(snap.Spec.VolGroup, getLVMSnapName(snap.Name))
	if !ok {
		logger.StdLog.Infof("lvm: snapshot %s does not exist, skipping deletion", snapVolume)
		return nil
	}

	if err != nil {
		logger.StdLog.Errorf("lvm: error checking for snapshot %s, error: %v", snapVolume, err)
		return err
	}

	args := buildLVMSnapDestroyArgs(snap)
	cmd := exec.Command(LVRemove, args...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		logger.StdLog.Errorf("lvm: could not remove snapshot %s cmd %v error: %s", snapVolume, args, string(out))
		return err
	}

	logger.StdLog.Infof("removed snapshot %s", snapVolume)
	return nil

}

// getSnapName is used to remove the snapshot prefix from the snapname. since names starting
// with "snapshot" are reserved in lvm2
func getLVMSnapName(snapName string) string {
	return strings.TrimPrefix(snapName, "snapshot-")
}
