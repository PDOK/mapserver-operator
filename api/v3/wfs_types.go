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

package v3

import (
	shared_model "github.com/pdok/smooth-operator/model"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:subresource:status
// versionName=v3
// +kubebuilder:resource:categories=pdok
// +kubebuilder:resource:path=wfs

// WFS is the Schema for the wfs API.
type WFS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WFSSpec                     `json:"spec,omitempty"`
	Status shared_model.OperatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WFSList contains a list of WFS.
type WFSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WFS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WFS{}, &WFSList{})
}

// WFSSpec vertegenwoordigt de hoofdstruct voor de YAML-configuratie
type WFSSpec struct {
	Lifecycle *shared_model.Lifecycle `json:"lifecycle"`
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// Optional strategic merge patch for the pod in the deployment. E.g. to patch the resources or add extra env vars.
	PodSpecPatch                 *corev1.PodSpec                            `json:"podSpecPatch,omitempty"`
	HorizontalPodAutoscalerPatch *autoscalingv2.HorizontalPodAutoscalerSpec `json:"horizontalPodAutoscalerPatch"`
	Options                      *Options                                   `json:"options"`
	Service                      WFSService                                 `json:"service"`
}

type WFSService struct {
	Prefix            string   `json:"prefix"`
	BaseURL           string   `json:"baseUrl"`
	Inspire           *Inspire `json:"inspire,omitempty"`
	Mapfile           *Mapfile `json:"mapfile,omitempty"`
	OwnerInfoRef      string   `json:"ownerInfoRef"`
	Title             string   `json:"title"`
	Abstract          string   `json:"abstract"`
	Keywords          []string `json:"keywords"`
	Fees              *string  `json:"fees,omitempty"`
	AccessConstraints string   `json:"accessConstraints"`
	DefaultCrs        string   `json:"defaultCrs"`
	OtherCrs          []string `json:"otherCrs,omitempty"`
	Bbox              *Bbox    `json:"bbox,omitempty"`
	// CountDefault -> wfs_maxfeatures in mapfile
	CountDefault *string       `json:"countDefault,omitempty"`
	FeatureTypes []FeatureType `json:"featureTypes"`
}

type Bbox struct {
	// EXTENT/wfs_extent in mapfile
	//nolint:tagliatelle
	DefaultCRS shared_model.BBox `json:"defaultCRS"`
}

type FeatureType struct {
	Name               string       `json:"name"`
	Title              string       `json:"title"`
	Abstract           string       `json:"abstract"`
	Keywords           []string     `json:"keywords"`
	DatasetMetadataURL MetadataURL  `json:"datasetMetadataUrl"`
	Bbox               *FeatureBbox `json:"bbox,omitempty"`
	Data               Data         `json:"data"`
}

type FeatureBbox struct {
	//nolint:tagliatelle
	DefaultCRS shared_model.BBox  `json:"defaultCRS"`
	WGS84      *shared_model.BBox `json:"wgs84,omitempty"`
}

func (wfs *WFS) HasPostgisData() bool {
	for _, featureType := range wfs.Spec.Service.FeatureTypes {
		if featureType.Data.Postgis != nil {
			return true
		}
	}
	return false
}
