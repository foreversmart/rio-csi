package mtypes

type Type string

const (
	TypeBlock      Type = "Block"
	TypeFileSystem Type = "FileSystem"
)

type Info struct {
	// +kubebuilder:validation:Optional
	MountType Type `json:"mount_type"`
	// +kubebuilder:validation:Optional
	VolumeInfo *VolumeInfo `json:"volume_info"`
	// +kubebuilder:validation:Optional
	PodInfo *PodInfo `json:"pod_info"`
}

// VolumeInfo contains the volume related info
// for all types of volumes in Volume
type VolumeInfo struct {
	// FSType of a volume will specify the
	// format type - ext4(default), xfs of PV
	// +kubebuilder:validation:Optional
	FSType string `json:"fsType"`

	// AccessMode of a volume will hold the
	// access mode of the volume
	// +kubebuilder:validation:Optional
	AccessModes []string `json:"accessModes"`

	// MountPath of the volume will hold the
	// path on which the volume is mounted
	// on that node
	// +kubebuilder:validation:Optional
	MountPath string `json:"mountPath"`

	// DevicePath is device path in the host
	// +kubebuilder:validation:Optional
	DevicePath string `json:"device_path"`

	// RawDevicePaths is all device path in the host device
	// +kubebuilder:validation:Optional
	RawDevicePaths []string `json:"raw_device_paths"`

	// MountOptions specifies the options with
	// which mount needs to be attempted
	// +kubebuilder:validation:Optional
	MountOptions []string `json:"mountOptions"`
}

// PodInfo contains the pod, LVGroup related info
type PodInfo struct {
	// UID is the Uid of the pod
	// +kubebuilder:validation:Optional
	UID string `json:"uid"`

	// Name is the Name of the pod
	// +kubebuilder:validation:Optional
	Name string `json:"name"`

	// Namespace is the namespace of the pod
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace"`

	// NodeId is the node id of the pod
	// +kubebuilder:validation:Optional
	NodeId string `json:"node_id"`
}
