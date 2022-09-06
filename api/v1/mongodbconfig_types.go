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

// MongoDBConfigSpec defines the desired state of MongoDBConfig
type MongoDBConfigSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`(?m)mongodb:\/\/(?:(?:[^:]+):(?:[^@]+)?@)?(?:(?:(?:[^\/]+)|(?:\/.+.sock?),?)+)(?:\/([^\/\."*<>:\|\?]*))?(?:\?(?:(.+=.+)&?)+)*`
	// MongoURL is a mongodb connection url
	MongoURL string `json:"mongourl,omitempty"`

	// +kubebuilder:validation:Required
	// Collection is a mongodb collection name
	Collection string `json:"collection,omitempty"`
}

// MongoDBConfigStatus defines the observed state of MongoDBConfig
type MongoDBConfigStatus struct {
	Ready      string             `json:"ready,omitempty"`
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.ready`,description=`Current state of the MongoDBConfig`
// +kubebuilder:resource:shortName=mdbc
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// MongoDBConfig is the Schema for the mongodbconfigs API
type MongoDBConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoDBConfigSpec   `json:"spec,omitempty"`
	Status MongoDBConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MongoDBConfigList contains a list of MongoDBConfig
type MongoDBConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDBConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MongoDBConfig{}, &MongoDBConfigList{})
}

type MongoDBConfigConditionType string

const (
	Ready               MongoDBConfigConditionType = "Ready"
	NoMongoURLSpecified MongoDBConfigConditionType = "NoMongoURLSpecified"
	ConnectError        MongoDBConfigConditionType = "ConnectError"
)
