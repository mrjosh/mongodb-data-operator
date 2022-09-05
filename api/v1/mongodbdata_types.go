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

// MongoDBDataSpec defines the desired state of MongoDBData
type MongoDBDataSpec struct {
	// DB is a MongoDBConfig CRD name
	DB string `json:"db,omitempty"`

	// Data is a MongodDB collection data
	Data MongoDBDataField `json:"data,omitempty"`
}

type MongoDBDataField struct {
	Firstname string `json:"firstname,omitempty"`
	Lastname  string `json:"lastname,omitempty"`
	Email     string `json:"email,omitempty"`
	Age       uint8  `json:"age,omitempty"`
}

// MongoDBDataStatus defines the observed state of MongoDBData
type MongoDBDataStatus struct {
	State string `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// MongoDBData is the Schema for the mongodbdata API
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="Current state of the MongoDBData"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC."
type MongoDBData struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoDBDataSpec   `json:"spec,omitempty"`
	Status MongoDBDataStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MongoDBDataList contains a list of MongoDBData
type MongoDBDataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDBData `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MongoDBData{}, &MongoDBDataList{})
}
