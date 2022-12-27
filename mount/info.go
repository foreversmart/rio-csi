package mount

// Info contains the volume related info
// for all types of volumes in Volume
type Info struct {
	// FSType of a volume will specify the
	// format type - ext4(default), xfs of PV
	FSType string `json:"fsType"`

	// AccessMode of a volume will hold the
	// access mode of the volume
	AccessModes []string `json:"accessModes"`

	// MountPath of the volume will hold the
	// path on which the volume is mounted
	// on that node
	MountPath string `json:"mountPath"`

	// MountOptions specifies the options with
	// which mount needs to be attempted
	MountOptions []string `json:"mountOptions"`
}

// PodInfo contains the pod, LVGroup related info
type PodInfo struct {
	// UID is the Uid of the pod
	UID string

	// Name is the Name of the pod
	Name string

	// Namespace is the namespace of the pod
	Namespace string

	// NodeId is the node id of the pod
	NodeId string
}
