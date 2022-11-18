/*
Copyright 2022.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VolumeSpec defines the desired state of Volume
type VolumeSpec struct {
	// OwnerNodeID is the Node ID where the volume group is present which is where
	// the volume has been provisioned.
	// OwnerNodeID can not be edited after the volume has been provisioned.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	OwnerNodeID string `json:"ownerNodeID"`

	// VolGroup specifies the name of the volume group where the volume has been created.
	// +kubebuilder:validation:Required
	VolGroup string `json:"volGroup"`

	// VgPattern specifies the regex to choose volume groups where volume
	// needs to be created.
	// +kubebuilder:validation:Required
	VgPattern string `json:"vgPattern"`

	// Capacity of the volume
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Capacity string `json:"capacity"`

	// Shared specifies whether the volume can be shared among multiple pods.
	// If it is not set to "yes", then the LVM LocalPV Driver will not allow
	// the volumes to be mounted by more than one pods.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=yes;no
	Shared string `json:"shared,omitempty"`

	// ThinProvision specifies whether logical volumes can be thinly provisioned.
	// If it is set to "yes", then the LVM LocalPV Driver will create
	// thinProvision i.e. logical volumes that are larger than the available extents.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=yes;no
	ThinProvision string `json:"thinProvision,omitempty"`

	// +kubebuilder:validation:Required
	Lun string `json:"lun"`
	// +kubebuilder:validation:Required
	IscsiBlock string `json:"iscsi_block"`
	// +kubebuilder:validation:Required
	Target string `json:"target"`
}

// VolumeStatus defines the observed state of Volume
type VolumeStatus struct {
	// State specifies the current state of the volume provisioning request.
	// The state "Pending" means that the volume creation request has not
	// processed yet. The state "Ready" means that the volume has been created
	// and it is ready for the use. "Failed" means that volume provisioning
	// has been failed and will not be retried by node agent controller.
	// +kubebuilder:validation:Enum=Pending;Ready;Failed
	State string `json:"state,omitempty"`

	// Error denotes the error occurred during provisioning/expanding a volume.
	// Error field should only be set when State becomes Failed.
	Error *VolumeError `json:"error,omitempty"`
}

// VolumeError specifies the error occurred during volume provisioning.
type VolumeError struct {
	Code    VolumeErrorCode `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
}

// VolumeErrorCode represents the error code to represent
// specific class of errors.
type VolumeErrorCode string

// +genclient

// Volume is the Schema for the volumes API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=vol
// +kubebuilder:printcolumn:name="VolGroup",type=string,JSONPath=`.spec.volGroup`,description="volume group where the volume is created"
// +kubebuilder:printcolumn:name="Node",type=string,JSONPath=`.spec.ownerNodeID`,description="Node where the volume is created"
// +kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.spec.capacity`,description="Size of the volume"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`,description="Status of the volume"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Age of the volume"
type Volume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeSpec   `json:"spec,omitempty"`
	Status VolumeStatus `json:"status,omitempty"`
}

// VolumeList contains a list of Volume
// +kubebuilder:object:root=true
type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Volume `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Volume{}, &VolumeList{})
}

const (
	// Internal represents system internal error.
	Internal VolumeErrorCode = "Internal"
	// InsufficientCapacity represent lvm vg doesn't
	// have enough capacity to fit the lv request.
	InsufficientCapacity VolumeErrorCode = "InsufficientCapacity"
)
