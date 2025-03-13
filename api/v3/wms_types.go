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
	"maps"
	"slices"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WMSSpec defines the desired state of WMS.
type WMSSpec struct {
	Lifecycle *shared_model.Lifecycle `json:"lifecycle"`

	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// Optional strategic merge patch for the pod in the deployment. E.g. to patch the resources or add extra env vars.
	PodSpecPatch                 *corev1.PodSpec                            `json:"podSpecPatch,omitempty"`
	HorizontalPodAutoscalerPatch *autoscalingv2.HorizontalPodAutoscalerSpec `json:"horizontalPodAutoscalerPatch"`
	Options                      *Options                                   `json:"options"`
	Service                      WMSService                                 `json:"service"`
}

type WMSService struct {
	BaseURL           string        `json:"baseUrl"`
	Title             string        `json:"title"`
	Abstract          string        `json:"abstract"`
	Keywords          []string      `json:"keywords"`
	OwnerInfoRef      string        `json:"ownerInfoRef"`
	Fees              *string       `json:"fees"`
	AccessConstraints string        `json:"accessConstraints"`
	MaxSize           int32         `json:"maxSize"`
	Inspire           *Inspire      `json:"inspire,omitempty"`
	DataEPSG          string        `json:"dataEPSG"`
	Resolution        float32       `json:"resolution"`
	DefResolution     float32       `json:"defResolution"`
	StylingAssets     StylingAssets `json:"stylingAssets"`
	Mapfile           *Mapfile      `json:"mapfile"`
	Layer             Layer         `json:"layer"`
}

type StylingAssets struct {
	BlobKeys      []string                      `json:"blobKeys"`
	ConfigMapRefs []corev1.ConfigMapKeySelector `json:"configMapRefs"`
}

type Layer struct {
	Name                string           `json:"name"`
	Title               string           `json:"title"`
	Abstract            string           `json:"abstract"`
	Keywords            []string         `json:"keywords"`
	BoundingBoxes       []WMSBoundingBox `json:"boundingBoxes"`
	Authority           Authority        `json:"authority"`
	DatasetMetadataURL  MetadataURL      `json:"datasetMetadataUrl"`
	MinScaleDenominator float32          `json:"minscaledenominator"`
	MaxScaleDenominator float32          `json:"maxscaledenominator"`
	Style               Style            `json:"style"`
	LabelNoClip         bool             `json:"labelNoClip"`
	Data                Data             `json:"data"`
	Layers              []Layer          `json:"layers"`
}

type WMSBoundingBox struct {
	CRS  string            `json:"crs"`
	BBox shared_model.BBox `json:"bbox"`
}

type Authority struct {
	Name                     string `json:"name"`
	URL                      string `json:"url"`
	SpatialDatasetIdentifier string `json:"spatialDatasetIdentifier"`
}

type Style struct {
	Name          string `json:"name"`
	Title         string `json:"title"`
	Abstract      string `json:"abstract"`
	Visualization string `json:"visualization"`
	Legend        Legend `json:"legend"`
}

type Legend struct {
	Width   int32  `json:"width"`
	Height  int32  `json:"height"`
	Format  string `json:"format"`
	BlobKey string `json:"blobKey"`
}

// WMSStatus defines the observed state of WMS.
type WMSStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:subresource:status
// versionName=v3
// +kubebuilder:resource:categories=pdok
// +kubebuilder:resource:path=wms

// WMS is the Schema for the wms API.
type WMS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WMSSpec                     `json:"spec,omitempty"`
	Status shared_model.OperatorStatus `json:"status,omitempty"`
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

func (wms *WMS) GetUniqueTiffBlobKeys() []string {
	blobKeys := map[string]bool{}

	if wms.Spec.Service.Layer.Data.TIF != nil && wms.Spec.Service.Layer.Data.TIF.BlobKey != "" {
		blobKeys[wms.Spec.Service.Layer.Data.TIF.BlobKey] = true
	}

	if len(wms.Spec.Service.Layer.Layers) > 0 {
		for _, layer := range wms.Spec.Service.Layer.Layers {
			if layer.Data.TIF != nil && layer.Data.TIF.BlobKey != "" {
				blobKeys[layer.Data.TIF.BlobKey] = true
			}
		}
	}
	return slices.Collect(maps.Keys(blobKeys))
}
