/*
MIT License

Copyright (c) 2024 Publieke Dienstverlening op de Kaart

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package v2beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WFS is the Schema for the wfs API.
type WFS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WFSSpec `json:"spec,omitempty"`
	Status *Status `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WFSList contains a list of WFS.
type WFSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WFS `json:"items"`
}

// WFSSpec is the struct for all fields defined in the WFS CRD
type WFSSpec struct {
	General    General       `json:"general"`
	Service    WFSService    `json:"service"`
	Kubernetes Kubernetes    `json:"kubernetes"`
	Options    WMSWFSOptions `json:"options"`
}

// WFSService is the struct with all service specific options
type WFSService struct {
	Title              string    `json:"title"`
	Inspire            bool      `json:"inspire"`
	Abstract           string    `json:"abstract"`
	AccessConstraints  string    `json:"accessConstraints"`
	Keywords           []string  `json:"keywords"`
	MetadataIdentifier string    `json:"metadataIdentifier"`
	Authority          Authority `json:"authority"`
	Extent             *string   `json:"extent,omitempty"`
	Maxfeatures        *string   `json:"maxfeatures,omitempty"`
	//nolint:tagliatelle
	DataEPSG     string        `json:"dataEPSG"`
	FeatureTypes []FeatureType `json:"featureTypes"`
	Mapfile      *Mapfile      `json:"mapfile,omitempty"`
}

// FeatureType is the struct for all feature type level fields
type FeatureType struct {
	Name                      string   `json:"name"`
	Title                     string   `json:"title"`
	Abstract                  string   `json:"abstract"`
	Keywords                  []string `json:"keywords"`
	DatasetMetadataIdentifier string   `json:"datasetMetadataIdentifier"`
	SourceMetadataIdentifier  string   `json:"sourceMetadataIdentifier"`
	Extent                    *string  `json:"extent,omitempty"`
	Data                      Data     `json:"data"`
}

func init() {
	SchemeBuilder.Register(&WFS{}, &WFSList{})
}
