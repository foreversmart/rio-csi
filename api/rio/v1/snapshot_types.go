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

// SnapshotSpec defines the desired state of Snapshot
type SnapshotSpec struct {
	// OwnerNodeID is the Node ID where the volume group is present which is where
	// the snapshot has been provisioned.
	// OwnerNodeID can not be edited after the snapshot has been provisioned.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	OwnerNodeID string `json:"ownerNodeID"`

	// VolGroup specifies the name of the volume group where the snapshot has been created.
	// +kubebuilder:validation:Required
	VolGroup string `json:"volGroup"`

	// SnapSize specifies the space reserved for the snapshot
	// +kubebuilder:validation:Required
	SnapSize string `json:"snapSize,omitempty"`
}

// SnapshotStatus defines the observed state of Snapshot
type SnapshotStatus struct {
	State string `json:"state,omitempty"`
}

// +genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Snapshot is the Schema for the snapshots API
type Snapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SnapshotSpec   `json:"spec,omitempty"`
	Status SnapshotStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=lvmsnapshots

// SnapshotList contains a list of Snapshot
type SnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Snapshot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Snapshot{}, &SnapshotList{})
}
