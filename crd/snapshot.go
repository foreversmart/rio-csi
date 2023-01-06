package crd

import (
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/logger"
)

// ProvisionSnapshot creates a Snapshot CR
func ProvisionSnapshot(snap *apis.Snapshot) error {
	_, err := client.DefaultClient.InternalClientSet.RioV1().Snapshots(RioNamespace).Create(context.Background(), snap, metav1.CreateOptions{})
	if err == nil {
		logger.StdLog.Infof("provosioned snapshot %s", snap.Name)
	}
	return err
}

// DeleteSnapshot deletes the LVMSnapshot CR
func DeleteSnapshot(snapName string) error {
	err := client.DefaultClient.InternalClientSet.RioV1().Snapshots(RioNamespace).Delete(context.Background(), snapName, metav1.DeleteOptions{})
	if err == nil {
		logger.StdLog.Infof("deprovisioned snapshot %s", snapName)
	}

	return err
}

// GetSnapshot fetches the given LVM snapshot
func GetSnapshot(snapID string) (*apis.Snapshot, error) {
	snap, err := client.DefaultClient.InternalClientSet.RioV1().Snapshots(RioNamespace).Get(context.Background(), snapID, metav1.GetOptions{})
	return snap, err
}

// GetSnapshotForVolume fetches all the snapshots for the given volume
func GetSnapshotForVolume(volumeID string) (*apis.SnapshotList, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: VolKey + "=" + volumeID,
	}
	snapList, err := client.DefaultClient.InternalClientSet.RioV1().Snapshots(RioNamespace).List(context.Background(), listOptions)
	return snapList, err
}

// GetSnapshotStatus returns the status of Snapshot
func GetSnapshotStatus(snapID string) (string, error) {
	getOptions := metav1.GetOptions{}
	snap, err := client.DefaultClient.InternalClientSet.RioV1().Snapshots(RioNamespace).Get(context.Background(), snapID, getOptions)
	if err != nil {
		logger.StdLog.Errorf("Get snapshot failed %s err: %s", snap.Name, err.Error())
		return "", err
	}
	return snap.Status.State, nil
}

// UpdateSnapInfo updates Snapshot CR with node id and finalizer
func UpdateSnapInfo(snap *apis.Snapshot, state string) (newSnap *apis.Snapshot, err error) {
	if snap.Finalizers != nil {
		return nil, nil
	}

	labels := map[string]string{
		NodeKey: NodeID,
	}
	snap.Labels = labels

	switch state {
	case StatusReady:
		finalizers := []string{RioFinalizer}
		snap.Finalizers = finalizers
	}

	newSnap, err = client.DefaultClient.InternalClientSet.RioV1().Snapshots(RioNamespace).Update(context.Background(), snap, metav1.UpdateOptions{})
	if err != nil {
		logger.StdLog.Error(err)
		return nil, err
	}

	// update snap status
	if newSnap.Status.State != state {
		newSnap.Status.State = state
		newSnap, err = client.DefaultClient.InternalClientSet.RioV1().Snapshots(RioNamespace).UpdateStatus(context.Background(), newSnap, metav1.UpdateOptions{})
	}

	return
}

// RemoveSnapFinalizer adds finalizer to Snapshot CR
func RemoveSnapFinalizer(snap *apis.Snapshot) (newSnap *apis.Snapshot, err error) {
	snap.Finalizers = nil

	newSnap, err = client.DefaultClient.InternalClientSet.RioV1().Snapshots(RioNamespace).Update(context.Background(), snap, metav1.UpdateOptions{})
	return
}
