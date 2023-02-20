package driver

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"qiniu.io/rio-csi/lib/lvm/common/errors"
	"strings"
)

// validateRequest validates if the requested service is
// supported by the driver
func (cs *ControllerServer) validateRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	for _, item := range cs.Driver.serviceCapabilities {
		if c == item.GetRpc().GetType() {
			return nil
		}
	}

	return status.Error(
		codes.InvalidArgument,
		fmt.Sprintf("failed to validate request: {%s} is not supported", c),
	)
}

func (cs *ControllerServer) validateVolumeCreateReq(req *csi.CreateVolumeRequest) error {
	err := cs.validateRequest(
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to handle create volume request for {%s}",
			req.GetName(),
		)
	}

	if req.GetName() == "" {
		return status.Error(
			codes.InvalidArgument,
			"failed to handle create volume request: missing volume name",
		)
	}

	volCapabilities := req.GetVolumeCapabilities()
	if volCapabilities == nil {
		return status.Error(
			codes.InvalidArgument,
			"failed to handle create volume request: missing volume capabilities",
		)
	}

	validateSupportedVolumeCapabilities := func(volCap *csi.VolumeCapability) error {
		// VolumeCapabilities will contain volume mode
		if mode := volCap.GetAccessMode(); mode != nil {
			inputMode := mode.GetMode()
			// At the moment we only support SINGLE_NODE_WRITER i.e Read-Write-Once
			var isModeSupported bool
			for _, supporteVolCapability := range cs.Driver.accessModes {
				if inputMode == supporteVolCapability.Mode {
					isModeSupported = true
					break
				}
			}

			if !isModeSupported {
				return status.Errorf(codes.InvalidArgument,
					"only ReadwriteOnce access mode is supported",
				)
			}
		}

		if volCap.GetBlock() == nil && volCap.GetMount() == nil {
			return status.Errorf(codes.InvalidArgument,
				"only Block mode (or) FileSystem mode is supported",
			)
		}

		return nil
	}

	for _, volCap := range volCapabilities {
		if err := validateSupportedVolumeCapabilities(volCap); err != nil {
			return err
		}
	}

	return nil
}

func (cs *ControllerServer) validateDeleteVolumeReq(req *csi.DeleteVolumeRequest) error {
	volumeID := strings.ToLower(req.GetVolumeId())
	if volumeID == "" {
		return status.Error(
			codes.InvalidArgument,
			"failed to handle delete volume request: missing volume id",
		)
	}

	// TODO snapshot check
	// volume should not be deleted if there are active snapshots present for the volume
	//snapList, err := lvm.GetSnapshotForVolume(volumeID)
	//
	//if err != nil {
	//	return status.Errorf(
	//		codes.NotFound,
	//		"failed to handle delete volume request for {%s}, "+
	//			"validation failed checking for active snapshots. Error: %s",
	//		req.VolumeId,
	//		err.Error(),
	//	)
	//}
	//
	//// delete is not supported if there are any snapshots present for the volume
	//if len(snapList.Items) != 0 {
	//	return status.Errorf(
	//		codes.Internal,
	//		"failed to handle delete volume request for {%s} with %d active snapshots",
	//		req.VolumeId,
	//		len(snapList.Items),
	//	)
	//}

	err := cs.validateRequest(
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to handle delete volume request for {%s} : validation failed",
			volumeID,
		)
	}
	return nil
}
