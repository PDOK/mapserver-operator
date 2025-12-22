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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true
// +kubebuilder:skipversion

// WMS is the Schema for the wms API.
type WMS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WMSSpec `json:"spec,omitempty"`
	Status *Status `json:"status,omitempty"`
}

// WMSSpec is the struct for all fields defined in the WMS CRD
type WMSSpec struct {
	General    General        `json:"general"`
	Service    WMSService     `json:"service"`
	Options    *WMSWFSOptions `json:"options,omitempty"`
	Kubernetes Kubernetes     `json:"kubernetes"`
}

// WMSService is the struct for all service level fields
type WMSService struct {
	Inspire  bool   `json:"inspire,omitempty"`
	Title    string `json:"title"`
	Abstract string `json:"abstract"`
	// +kubebuilder:default="https://creativecommons.org/publicdomain/zero/1.0/deed.nl"
	AccessConstraints  *string    `json:"accessConstraints,omitempty"` // Pointer for CRD conversion as defaulting is not applied there
	Keywords           []string   `json:"keywords"`
	MetadataIdentifier string     `json:"metadataIdentifier"`
	Authority          Authority  `json:"authority"`
	Layers             []WMSLayer `json:"layers"`
	//nolint:tagliatelle
	DataEPSG      string         `json:"dataEPSG"`
	Extent        *string        `json:"extent,omitempty"`
	Maxsize       *float64       `json:"maxSize,omitempty"`
	Resolution    *int           `json:"resolution,omitempty"`
	DefResolution *int           `json:"defResolution,omitempty"`
	StylingAssets *StylingAssets `json:"stylingAssets,omitempty"`
	Mapfile       *Mapfile       `json:"mapfile,omitempty"`
}

// WMSLayer is the struct for all layer level fields
type WMSLayer struct {
	Name                      string   `json:"name"`
	Group                     *string  `json:"group,omitempty"`
	Visible                   *bool    `json:"visible,omitempty"`
	Title                     *string  `json:"title,omitempty"`
	Abstract                  *string  `json:"abstract,omitempty"`
	Keywords                  []string `json:"keywords,omitempty"`
	DatasetMetadataIdentifier *string  `json:"datasetMetadataIdentifier,omitempty"`
	SourceMetadataIdentifier  *string  `json:"sourceMetadataIdentifier,omitempty"`
	Styles                    []Style  `json:"styles"`
	Extent                    *string  `json:"extent,omitempty"`
	MinScale                  *float64 `json:"minScale,omitempty"`
	MaxScale                  *float64 `json:"maxScale,omitempty"`
	LabelNoClip               bool     `json:"labelNoClip,omitempty"`
	Data                      *Data    `json:"data,omitempty"`
}

// Style is the struct for all style level fields
type Style struct {
	Name          string      `json:"name"`
	Title         *string     `json:"title,omitempty"`
	Abstract      *string     `json:"abstract,omitempty"`
	Visualization *string     `json:"visualization,omitempty"`
	LegendFile    *LegendFile `json:"legendFile,omitempty"`
}

// LegendFile is the struct containing the location of the legendfile
type LegendFile struct {
	BlobKey string `json:"blobKey"`
}

// StylingAssets is the struct containing the location of styling assets
type StylingAssets struct {
	ConfigMapRefs []ConfigMapRef `json:"configMapRefs,omitempty"`
	BlobKeys      []string       `json:"blobKeys,omitempty"`
}

// ConfigMapRef contains all the config map name and all keys in that mapserver that are relevant
// the Keys can be empty, so that the v1 WMS can convert to the v2beta1 WMS
type ConfigMapRef struct {
	Name string   `json:"name"`
	Keys []string `json:"keys,omitempty"`
}

// Mapfile contains the ConfigMapKeyRef containing a mapfile
type Mapfile struct {
	ConfigMapKeyRef corev1.ConfigMapKeySelector `json:"configMapKeyRef"`
}

// +kubebuilder:object:root=true

// WMSList contains a list of WMS.
type WMSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WMS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WMS{}, &WMSList{})
}
