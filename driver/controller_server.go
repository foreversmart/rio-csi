package driver

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/mount"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/driver/dparams"
	"qiniu.io/rio-csi/enums"
	"qiniu.io/rio-csi/lib/iscsi"
	"qiniu.io/rio-csi/lib/lvm/builder/volbuilder"
	"qiniu.io/rio-csi/lib/lvm/common/errors"
	"qiniu.io/rio-csi/logger"
	"strconv"
	"strings"
)

type ControllerServer struct {
	Driver *RioCSI
	// Users add fields as needed.
	//
	// In the NFS CSI implementation, we need to mount the nfs server to the local,
	// so we need a mounter instance.
	//
	// In the CSI implementation of other storage vendors, you may need to add other
	// instances, such as the api client of Alibaba Cloud Storage.
	mounter mount.Interface
}

func (cs *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	logger.StdLog.Infof("received request to create volume %s", req.GetName())

	params, err := dparams.NewVolumeParams(req.GetParameters())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			"failed to parse csi volume params: %v", err)
	}

	logger.StdLog.Info("create volume parameters:", req.GetParameters())

	volName := strings.ToLower(req.GetName())
	size := getRoundedCapacity(req.GetCapacityRange().RequiredBytes)
	capacity := strconv.FormatInt(size, 10)
	volumeSource := req.GetVolumeContentSource()

	existVol, err := crd.GetVolume(volName)
	if err != nil {
		if !k8serror.IsNotFound(err) {
			logger.StdLog.Error(err)
			return nil, status.Errorf(codes.Aborted,
				"failed get lvm volume %v: %v", volName, err.Error())
		}

		existVol, err = nil, nil
	}

	// if exist return success
	if existVol != nil {
		topology := map[string]string{crd.TopologyKey: existVol.Spec.OwnerNodeID}
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				VolumeId:      volName,
				CapacityBytes: size,
				AccessibleTopology: []*csi.Topology{{
					Segments: topology,
				},
				},
				VolumeContext: map[string]string{crd.VolGroupKey: existVol.Spec.VolGroup},
				ContentSource: volumeSource,
			},
		}, nil
	}

	// TODO Schedule
	node := "xs2298"

	// TODO scheduler
	newVol, buildErr := volbuilder.NewBuilder().
		WithName(volName).
		WithCapacity(capacity).
		WithVgPattern(params.VgPattern.String()).
		WithOwnerNode(node).
		WithVolumeStatus(crd.StatusPending).
		WithShared(params.Shared).
		WithThinProvision(params.ThinProvision).Build()
	// set default iscsi lun is -1 means no lun device
	newVol.Spec.IscsiLun = -1

	if buildErr != nil {
		return nil, status.Error(codes.Internal, buildErr.Error())
	}

	cntx := map[string]string{crd.VolGroupKey: newVol.Spec.VolGroup}

	if volumeSource != nil {
		switch volumeSource.Type.(type) {
		case *csi.VolumeContentSource_Snapshot:
			if snapshot := volumeSource.GetSnapshot(); snapshot != nil {
				cntx["dataSource"] = snapshot.SnapshotId
				newVol.Spec.DataSource = snapshot.SnapshotId
				newVol.Spec.DataSourceType = enums.DataSourceTypeSnapshot
			}
		case *csi.VolumeContentSource_Volume:
			return nil, status.Error(codes.Unimplemented, "")
		}
	}

	// TODO mark volume for leak protection if pvc gets deleted
	// before the creation of pv.

	newVol, err = crd.ProvisionVolume(newVol)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "not able to provision the volume %s", err.Error())
	}

	// Wait Volume ready
	if newVol.Status.State == crd.StatusPending {
		if newVol, err = crd.WaitForVolumeProcessed(ctx, newVol.GetName()); err != nil {
			return nil, err
		}
	}

	//
	topology := map[string]string{crd.TopologyKey: newVol.Spec.OwnerNodeID}
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volName,
			CapacityBytes: size,
			AccessibleTopology: []*csi.Topology{{
				Segments: topology,
			},
			},
			VolumeContext: cntx,
			ContentSource: volumeSource,
		},
	}, nil
}

func (cs *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	logger.StdLog.Debugf("running DeleteVolume...")
	var err error
	if err = cs.validateDeleteVolumeReq(req); err != nil {
		return nil, err
	}

	volumeID := strings.ToLower(req.GetVolumeId())
	logger.StdLog.Infof("received request to delete volume %q", volumeID)
	vol, err := crd.GetVolume(volumeID)
	if err != nil {
		if k8serror.IsNotFound(err) {
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, errors.Wrapf(err, "failed to get volume for {%s}", volumeID)
	}

	// if volume is not already triggered for deletion, delete the volume.
	// otherwise, just wait for the existing deletion operation to complete.
	if vol.GetDeletionTimestamp() == nil {
		_, err = iscsi.UnmountLun(vol.Spec.IscsiTarget, fmt.Sprintf("%d", vol.Spec.IscsiLun))
		if err != nil {
			logger.StdLog.Error(volumeID, err)
			return nil, errors.Wrapf(err, "UnmountLun for {%s}", volumeID)
		}

		_, err = iscsi.UnPublicBlockDevice(volumeID)
		if err != nil {
			logger.StdLog.Error(volumeID, err)
			return nil, errors.Wrapf(err, "UnPublicBlockDevice for {%s}", volumeID)
		}

		err = iscsi.DeleteTarget(vol.Spec.IscsiTarget)
		if err != nil {
			logger.StdLog.Error(volumeID, err)
			return nil, errors.Wrapf(err, "DeleteTarget for {%s}", volumeID)
		}

		if err = crd.DeleteVolume(volumeID); err != nil {
			logger.StdLog.Error(volumeID, err)
			return nil, errors.Wrapf(err,
				"failed to handle delete volume request for {%s}", volumeID)
		}
	}

	if err = crd.WaitForVolumeDestroy(ctx, volumeID); err != nil {
		return nil, err
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *ControllerServer) ControllerPublishVolume(_ context.Context, _ *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	logger.StdLog.Debugf("running ControllerPublishVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented ControllerPublishVolume")
}

func (cs *ControllerServer) ControllerUnpublishVolume(_ context.Context, _ *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	logger.StdLog.Debugf("running ControllerUnpublishVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented ControllerUnpublishVolume")
}

func (cs *ControllerServer) ValidateVolumeCapabilities(_ context.Context, _ *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	logger.StdLog.Debugf("running ValidateVolumeCapabilities...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented ValidateVolumeCapabilities")
}

func (cs *ControllerServer) ListVolumes(_ context.Context, _ *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	logger.StdLog.Debugf("running ListVolumes...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented ListVolumes")
}

func (cs *ControllerServer) GetCapacity(_ context.Context, _ *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	logger.StdLog.Debugf("running GetCapacity...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented GetCapacity")
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities
func (cs *ControllerServer) ControllerGetCapabilities(_ context.Context, _ *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	logger.StdLog.Infof("get ControllerGetCapabilities")
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.Driver.serviceCapabilities,
	}, nil
}

func (cs *ControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	logger.StdLog.Infof("received request to create snapshot from volume %s", req.SourceVolumeId)
	snapshotName := strings.ToLower(req.GetName())
	snap, err := crd.GetSnapshot(snapshotName)
	if err != nil {
		if !k8serror.IsNotFound(err) {
			logger.StdLog.Error(err)
			return nil, status.Errorf(codes.Aborted, "failed get snapshot %s: error %v", snapshotName, err)
		}

		snap, err = nil, nil
	}

	if snap != nil {
		logger.StdLog.Errorf("snapshot %s already present", snapshotName)
		return &csi.CreateSnapshotResponse{
			Snapshot: &csi.Snapshot{
				SnapshotId:     snap.Name,
				SourceVolumeId: req.SourceVolumeId,
			},
		}, nil
	}

	snapshot := &apis.Snapshot{}

	vol, err := crd.GetVolume(req.SourceVolumeId)
	if err != nil {
		logger.StdLog.Error(err)
		return nil, err
	}

	sizeBytes, err := strconv.ParseInt(vol.Spec.Capacity, 10, 64)
	if err != nil {
		logger.StdLog.Errorf("cant parse vol %s capacity %s", vol.Name, vol.Spec.Capacity)
		return nil, err
	}

	// TODO control snapshot snapshot size
	snapshot.Spec.SnapSize = vol.Spec.Capacity
	snapshot.Spec.VolGroup = vol.Spec.VolGroup
	snapshot.Spec.VolGroup = vol.Spec.VolGroup
	snapshot.Spec.OwnerNodeID = vol.Spec.OwnerNodeID
	snapshot.Name = snapshotName

	labels := map[string]string{
		crd.VolKey: vol.Name,
	}

	crd.WithLabels(snapshot, labels)

	err = crd.ProvisionSnapshot(snapshot)
	if err != nil {
		logger.StdLog.Error(err)
		return nil, err
	}

	// TODO ready to use vsc when snapshot is ready
	return &csi.CreateSnapshotResponse{
		Snapshot: &csi.Snapshot{
			SizeBytes:      sizeBytes,
			SnapshotId:     snapshot.Name,
			SourceVolumeId: req.SourceVolumeId,
			CreationTime:   timestamppb.Now(),
			ReadyToUse:     true,
		},
	}, nil
}

func (cs *ControllerServer) DeleteSnapshot(_ context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	snapshotID := strings.ToLower(req.GetSnapshotId())
	logger.StdLog.Infof("received request to delete snapshot", snapshotID)

	snapshot, err := crd.GetSnapshot(snapshotID)
	if err != nil {
		if k8serror.IsNotFound(err) {
			return nil, nil
		}

		logger.StdLog.Errorf("GetSnapshot %s error %v", snapshotID, err)

		return nil, errors.Wrapf(err, "failed to get snapshot %s", snapshotID)
	}

	// if snapshot is not already triggered for deletion, delete the snapshot.
	// otherwise, just wait for the existing deletion operation to complete.
	if snapshot.GetDeletionTimestamp() == nil {
		err = crd.DeleteSnapshot(snapshotID)
		if err != nil {
			logger.StdLog.Error(snapshotID, err)
			return nil, errors.Wrapf(err, "failed to handle delete volume request for %s", snapshotID)
		}
	}

	return &csi.DeleteSnapshotResponse{}, nil
}

func (cs *ControllerServer) ListSnapshots(_ context.Context, _ *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	logger.StdLog.Debugf("running ListSnapshots...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented ListSnapshots")
}

func (cs *ControllerServer) ControllerExpandVolume(_ context.Context, _ *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	logger.StdLog.Debugf("running ControllerExpandVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented ControllerExpandVolume")
}

func (cs *ControllerServer) ControllerGetVolume(_ context.Context, _ *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	logger.StdLog.Debugf("running ControllerGetVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented ControllerGetVolume")
}
