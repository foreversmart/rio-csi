package driver

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/mount"
	"qiniu.io/rio-csi/iscsi"
	"qiniu.io/rio-csi/logger"
	"qiniu.io/rio-csi/lvm"
	"qiniu.io/rio-csi/lvm/builder/volbuilder"
	"qiniu.io/rio-csi/lvm/common/errors"
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
	logger.StdLog.Debugf("running CreateLVMVolume...")

	params, err := NewVolumeParams(req.GetParameters())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			"failed to parse csi volume params: %v", err)
	}
	logger.StdLog.Info("create volume parameters:", req.GetParameters())

	volName := strings.ToLower(req.GetName())
	size := getRoundedCapacity(req.GetCapacityRange().RequiredBytes)
	capacity := strconv.FormatInt(size, 10)

	vol, err := lvm.GetVolume(volName)
	if err != nil {
		if !k8serror.IsNotFound(err) {
			return nil, status.Errorf(codes.Aborted,
				"failed get lvm volume %v: %v", volName, err.Error())
		}
		vol, err = nil, nil
	}

	// TODO
	if vol != nil {
		return nil, status.Errorf(codes.AlreadyExists,
			"volume %s already present", volName)
	}

	// TODO Schedule Node
	node := "xs2298"
	contentSource := req.GetVolumeContentSource()
	if contentSource != nil && contentSource.GetSnapshot() != nil {
		return nil, status.Error(codes.Unimplemented, "")
	} else if contentSource != nil && contentSource.GetVolume() != nil {
		return nil, status.Error(codes.Unimplemented, "")
	} else {
		// TODO mark volume for leak protection if pvc gets deleted
		// before the creation of pv.

		// TODO scheduler
		volObj, buildErr := volbuilder.NewBuilder().
			WithName(volName).
			WithCapacity(capacity).
			WithVgPattern(params.VgPattern.String()).
			WithOwnerNode(node).
			WithVolumeStatus(lvm.LVMStatusPending).
			WithShared(params.Shared).
			WithThinProvision(params.ThinProvision).Build()
		// set default iscsi lun is -1 means no lun device
		volObj.Spec.IscsiLun = -1

		if buildErr != nil {
			return nil, status.Error(codes.Internal, buildErr.Error())
		}

		vol, err = lvm.ProvisionVolume(volObj)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "not able to provision the volume %s", err.Error())
		}

		// Wait Volume ready
		if vol.Status.State == lvm.LVMStatusPending {
			if vol, err = lvm.WaitForVolumeProcessed(ctx, vol.GetName()); err != nil {
				return nil, err
			}
		}
	}

	//
	cntx := map[string]string{lvm.VolGroupKey: vol.Spec.VolGroup}
	topology := map[string]string{lvm.LVMTopologyKey: vol.Spec.OwnerNodeID}
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volName,
			CapacityBytes: size,
			AccessibleTopology: []*csi.Topology{{
				Segments: topology,
			},
			},
			VolumeContext: cntx,
			ContentSource: contentSource,
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
	vol, err := lvm.GetVolume(volumeID)
	if err != nil {
		if k8serror.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get volume for {%s}", volumeID)
	}

	// if volume is not already triggered for deletion, delete the volume.
	// otherwise, just wait for the existing deletion operation to complete.
	if vol.GetDeletionTimestamp() == nil {
		_, err = iscsi.UnmountLun(vol.Spec.IscsiTarget, fmt.Sprintf("%d", vol.Spec.IscsiLun))
		if err != nil {
			logger.StdLog.Error(volumeID, err)
			return nil, errors.Wrapf(err, "failed to handle delete volume request for {%s}", volumeID)
		}

		_, err = iscsi.UnPublicBlockDevice(vol.Spec.IscsiTarget, volumeID)
		if err != nil {
			logger.StdLog.Error(volumeID, err)
			return nil, errors.Wrapf(err, "failed to handle delete volume request for {%s}", volumeID)
		}

		if err = lvm.DeleteVolume(volumeID); err != nil {
			logger.StdLog.Error(volumeID, err)
			return nil, errors.Wrapf(err,
				"failed to handle delete volume request for {%s}", volumeID)
		}
	}

	if err = lvm.WaitForVolumeDestroy(ctx, volumeID); err != nil {
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
	logger.StdLog.Debugf("running CreateSnapshot...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented CreateSnapshot")
}

func (cs *ControllerServer) DeleteSnapshot(_ context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	logger.StdLog.Debugf("running DeleteSnapshot...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented DeleteSnapshot")
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
