package driver

import (
	"fmt"
	"regexp"
)

// scheduling algorithm constants
const (
	// pick the node where less volumes are provisioned for the given volume group
	VolumeWeighted = "VolumeWeighted"

	// pick the node where total provisioned volumes have occupied less capacity from the given volume group
	CapacityWeighted = "CapacityWeighted"

	// pick the node which is less loaded space wise
	// this will be the default scheduler when none provided
	SpaceWeighted = "SpaceWeighted"
)

// VolumeParams holds collection of supported settings that can
// be configured in storage class.
type VolumeParams struct {
	// VgPattern specifies vg regex to use for
	// provisioning logical volumes.
	VgPattern *regexp.Regexp

	Scheduler     string
	Shared        string
	ThinProvision string
	// extra optional metadata passed by external provisioner
	// if enabled. See --extra-create-metadata flag for more details.
	// https://github.com/kubernetes-csi/external-provisioner#recommended-optional-arguments
	PVCName      string
	PVCNamespace string
	PVName       string
}

// NewVolumeParams parses the input params and instantiates new VolumeParams.
func NewVolumeParams(m map[string]string) (*VolumeParams, error) {
	params := &VolumeParams{ // set up defaults, if any.
		Scheduler:     SpaceWeighted,
		Shared:        "no",
		ThinProvision: "no",
	}
	// parameter keys may be mistyped from the CRD specification when declaring
	// the storageclass, which kubectl validation will not catch. Because
	// parameter keys (not values!) are all lowercase, keys may safely be forced
	// to the lower case.
	m = GetCaseInsensitiveMap(&m)

	// for ensuring backward compatibility, we first check if
	// there is any volgroup param exists for storage class.

	vgPattern := m["vgpattern"]
	volGroup, ok := m["volgroup"]
	if ok {
		vgPattern = fmt.Sprintf("^%v$", volGroup)
	}

	var err error
	if params.VgPattern, err = regexp.Compile(vgPattern); err != nil {
		return nil, fmt.Errorf("invalid volgroup/vgpattern param %v: %v", vgPattern, err)
	}

	// parse string params
	stringParams := map[string]*string{
		"scheduler":     &params.Scheduler,
		"shared":        &params.Shared,
		"thinprovision": &params.ThinProvision,
	}
	for key, param := range stringParams {
		value, ok := m[key]
		if !ok {
			continue
		}
		*param = value
	}

	params.PVCName = m["csi.storage.k8s.io/pvc/name"]
	params.PVCNamespace = m["csi.storage.k8s.io/pvc/namespace"]
	params.PVName = m["csi.storage.k8s.io/pv/name"]

	return params, nil
}
