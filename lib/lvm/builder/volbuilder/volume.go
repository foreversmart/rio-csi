/*
Copyright 2020 The OpenEBS Authors

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

package volbuilder

import (
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/lib/lvm/common/errors"
)

// Builder is the builder object for Volume
type Builder struct {
	volume *Volume
	errs   []error
}

// Volume is a wrapper over
// Volume API instance
type Volume struct {
	// Volume object
	Object *apis.Volume
}

// From returns a new instance of
// lvm volume
func From(vol *apis.Volume) *Volume {
	return &Volume{
		Object: vol,
	}
}

// NewBuilder returns new instance of Builder
func NewBuilder() *Builder {
	return &Builder{
		volume: &Volume{
			Object: &apis.Volume{},
		},
	}
}

// BuildFrom returns new instance of Builder
// from the provided api instance
func BuildFrom(volume *apis.Volume) *Builder {
	if volume == nil {
		b := NewBuilder()
		b.errs = append(
			b.errs,
			errors.New("failed to build volume object: nil volume"),
		)
		return b
	}
	return &Builder{
		volume: &Volume{
			Object: volume,
		},
	}
}

// WithNamespace sets the namespace of  Volume
func (b *Builder) WithNamespace(namespace string) *Builder {
	if namespace == "" {
		b.errs = append(
			b.errs,
			errors.New(
				"failed to build lvm volume object: missing namespace",
			),
		)
		return b
	}
	b.volume.Object.Namespace = namespace
	return b
}

// WithName sets the name of Volume
func (b *Builder) WithName(name string) *Builder {
	if name == "" {
		b.errs = append(
			b.errs,
			errors.New(
				"failed to build lvm volume object: missing name",
			),
		)
		return b
	}
	b.volume.Object.Name = name
	return b
}

// WithCapacity sets the Capacity of lvm volume by converting string
// capacity into Quantity
func (b *Builder) WithCapacity(capacity string) *Builder {
	if capacity == "" {
		b.errs = append(
			b.errs,
			errors.New(
				"failed to build lvm volume object: missing capacity",
			),
		)
		return b
	}
	b.volume.Object.Spec.Capacity = capacity
	return b
}

// WithOwnerNode sets owner node for the Volume where the volume should be provisioned
func (b *Builder) WithOwnerNode(host string) *Builder {
	b.volume.Object.Spec.OwnerNodeID = host
	return b
}

// WithVolumeStatus sets Volume status
func (b *Builder) WithVolumeStatus(status string) *Builder {
	b.volume.Object.Status.State = status
	return b
}

// WithShared sets where filesystem is shared or not
func (b *Builder) WithShared(shared string) *Builder {
	b.volume.Object.Spec.Shared = shared
	return b
}

// WithThinProvision sets where thinProvision is enable or not
func (b *Builder) WithThinProvision(thinProvision string) *Builder {
	b.volume.Object.Spec.ThinProvision = thinProvision
	return b
}

// WithVolGroup sets volume group name for creating volume
func (b *Builder) WithVolGroup(vg string) *Builder {
	if vg == "" {
		b.errs = append(
			b.errs,
			errors.New(
				"failed to build lvm volume object: missing vg name",
			),
		)
		return b
	}
	b.volume.Object.Spec.VolGroup = vg
	return b
}

// WithVgPattern sets volume group regex pattern.
func (b *Builder) WithVgPattern(pattern string) *Builder {
	if pattern == "" {
		b.errs = append(
			b.errs,
			errors.New(
				"failed to build lvm volume object: missing vg name",
			),
		)
		return b
	}
	b.volume.Object.Spec.VgPattern = pattern
	return b
}

// WithNodeName sets NodeID for creating the volume
func (b *Builder) WithNodeName(name string) *Builder {
	if name == "" {
		b.errs = append(
			b.errs,
			errors.New(
				"failed to build lvm volume object: missing node name",
			),
		)
		return b
	}
	b.volume.Object.Spec.OwnerNodeID = name
	return b
}

// WithLabels merges existing labels if any
// with the ones that are provided here
func (b *Builder) WithLabels(labels map[string]string) *Builder {
	if len(labels) == 0 {
		return b
	}

	if b.volume.Object.Labels == nil {
		b.volume.Object.Labels = map[string]string{}
	}

	for key, value := range labels {
		b.volume.Object.Labels[key] = value
	}
	return b
}

// WithFinalizer sets Finalizer name creating the volume
func (b *Builder) WithFinalizer(finalizer []string) *Builder {
	b.volume.Object.Finalizers = append(b.volume.Object.Finalizers, finalizer...)
	return b
}

// Build returns Volume API object
func (b *Builder) Build() (*apis.Volume, error) {
	if len(b.errs) > 0 {
		return nil, errors.Errorf("%+v", b.errs)
	}

	return b.volume.Object, nil
}
