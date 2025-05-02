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
	autoscalingv2 "k8s.io/api/autoscaling/v2"
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
	// Optional lifecycle settings
	Lifecycle *shared_model.Lifecycle `json:"lifecycle,omitempty"`

	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// Optional strategic merge patch for the pod in the deployment. E.g. to patch the resources or add extra env vars.
	PodSpecPatch                 *corev1.PodSpec                            `json:"podSpecPatch,omitempty"`
	HorizontalPodAutoscalerPatch *autoscalingv2.HorizontalPodAutoscalerSpec `json:"horizontalPodAutoscalerPatch,omitempty"`
	Options                      Options                                    `json:"options,omitempty"`

	// service configuration
	Service WFSService `json:"service"`
}

type WFSService struct {
	// Geonovum subdomein
	// +kubebuilder:validation:MinLength:=1
	Prefix string `json:"prefix"`

	// URL of the service
	// +kubebuilder:validation:Pattern:=`^https?://.*$`
	// +kubebuilder:validation:MinLength:=1
	URL string `json:"url"`

	// Config for Inspire services
	Inspire *Inspire `json:"inspire,omitempty"`

	// External Mapfile reference
	Mapfile *Mapfile `json:"mapfile,omitempty"`

	// Reference to OwnerInfo CR
	// +kubebuilder:validation:MinLength:=1
	OwnerInfoRef string `json:"ownerInfoRef"`

	// Service title
	// +kubebuilder:validation:MinLength:=1
	Title string `json:"title"`

	// Service abstract
	// +kubebuilder:validation:MinLength:=1
	Abstract string `json:"abstract"`

	// Keywords for capabilities
	// +kubebuilder:validation:MinItems:=1
	Keywords []string `json:"keywords"`

	// Optional Fees
	// +kubebuilder:validation:MinLength:=1
	Fees *string `json:"fees,omitempty"`

	// AccessConstraints URL
	// +kubebuilder:validation:Pattern:="https?://"
	// +kubebuilder:default="https://creativecommons.org/publicdomain/zero/1.0/deed.nl"

	// +kubebuilder:validation:MinLength:=1
	AccessConstraints string `json:"accessConstraints"`

	// Default CRS (DataEPSG)
	// +kubebuilder:validation:Pattern:="^EPSG:(28992|25831|25832|3034|3035|3857|4258|4326)$"
	// +kubebuilder:validation:MinLength:=1
	DefaultCrs string `json:"defaultCrs"`

	// Other supported CRS
	// +kubebuilder:validation:MinItems:=1
	OtherCrs []string `json:"otherCrs,omitempty"`

	// Service bounding box
	Bbox *Bbox `json:"bbox,omitempty"`

	// CountDefault -> wfs_maxfeatures in mapfile
	// +kubebuilder:validation:MinLength:=1
	CountDefault *string `json:"countDefault,omitempty"`

	// FeatureTypes configurations
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:Type=array
	FeatureTypes []FeatureType `json:"featureTypes"`
}

type Bbox struct {
	// EXTENT/wfs_extent in mapfile
	//nolint:tagliatelle
	// +kubebuilder:validation:Type=object
	DefaultCRS shared_model.BBox `json:"defaultCRS"`
}

// FeatureType defines a WFS feature
type FeatureType struct {
	// Name of the feature
	// +kubebuilder:validation:MinLength:=1
	Name string `json:"name"`

	// Title of the feature
	// +kubebuilder:validation:MinLength:=1
	Title string `json:"title"`

	// Abstract of the feature
	// +kubebuilder:validation:MinLength:=1
	Abstract string `json:"abstract"`

	// Keywords of the feature
	// +kubebuilder:validation:MinItems:=1
	Keywords []string `json:"keywords"`

	// Metadata URL
	// +kubebuilder:validation:Type=object
	DatasetMetadataURL MetadataURL `json:"datasetMetadataUrl"`

	// Optional feature bbox
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type:=object
	Bbox *FeatureBbox `json:"bbox,omitempty"`

	// FeatureType data connection
	// +kubebuilder:validation:Type=object
	Data Data `json:"data"`
}

// FeatureType bounding box, if provided it overrides the default extent
type FeatureBbox struct {
	// DefaultCRS defines the feature’s bounding box in the service’s own CRS
	//nolint:tagliatelle
	// +kubebuilder:validation:Type=object
	DefaultCRS shared_model.BBox `json:"defaultCRS"`

	// WGS84, if provided, gives the same bounding box reprojected into EPSG:4326.
	// +kubebuilder:validation:Type=object
	WGS84 *shared_model.BBox `json:"wgs84,omitempty"`
}

func (wfs *WFS) HasPostgisData() bool {
	for _, featureType := range wfs.Spec.Service.FeatureTypes {
		if featureType.Data.Postgis != nil {
			return true
		}
	}
	return false
}

func (wfs *WFS) Mapfile() *Mapfile {
	return wfs.Spec.Service.Mapfile
}

func (wfs *WFS) Type() ServiceType {
	return ServiceTypeWFS
}

func (wfs *WFS) PodSpecPatch() *corev1.PodSpec {
	return wfs.Spec.PodSpecPatch
}

func (wfs *WFS) HorizontalPodAutoscalerPatch() *autoscalingv2.HorizontalPodAutoscalerSpec {
	return wfs.Spec.HorizontalPodAutoscalerPatch
}

func (wfs *WFS) Options() *Options {
	return &wfs.Spec.Options
}

func (wfs *WFS) ID() string {
	return Sha1HashOfName(wfs)
}

func (wfs *WFS) URLPath() string {
	return wfs.Spec.Service.URL
}

func (wfs *WFS) GeoPackages() []*Gpkg {
	gpkgs := make([]*Gpkg, 0)

	for _, ft := range wfs.Spec.Service.FeatureTypes {
		if ft.Data.Gpkg != nil {
			gpkgs = append(gpkgs, ft.Data.Gpkg)
		}
	}

	return gpkgs
}
