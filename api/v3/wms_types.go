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
	"maps"
	"slices"
	"sort"
)

const (
	TopLayer   = "topLayer"
	DataLayer  = "dataLayer"
	GroupLayer = "groupLayer"
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
	Options                      Options                                    `json:"options,omitempty"`
	Service                      WMSService                                 `json:"service"`
}

type WMSService struct {
	URL          string   `json:"url"`
	Title        string   `json:"title"`
	Abstract     string   `json:"abstract"`
	Keywords     []string `json:"keywords"`
	OwnerInfoRef string   `json:"ownerInfoRef"`
	Fees         *string  `json:"fees,omitempty"`
	// +kubebuilder:default="https://creativecommons.org/publicdomain/zero/1.0/deed.nl"
	AccessConstraints string         `json:"accessConstraints"`
	MaxSize           *int32         `json:"maxSize,omitempty"`
	Inspire           *Inspire       `json:"inspire,omitempty"`
	DataEPSG          string         `json:"dataEPSG"`
	Resolution        *int32         `json:"resolution,omitempty"`
	DefResolution     *int32         `json:"defResolution,omitempty"`
	StylingAssets     *StylingAssets `json:"stylingAssets,omitempty"`
	Mapfile           *Mapfile       `json:"mapfile,omitempty"`
	Layer             Layer          `json:"layer"`
}

type StylingAssets struct {
	BlobKeys      []string       `json:"blobKeys"`
	ConfigMapRefs []ConfigMapRef `json:"configMapRefs"`
}

type ConfigMapRef struct {
	Name string   `json:"name"`
	Keys []string `json:"keys,omitempty"`
}

type Layer struct {
	Name                *string          `json:"name"`
	Title               *string          `json:"title,omitempty"`
	Abstract            *string          `json:"abstract,omitempty"`
	Keywords            []string         `json:"keywords"`
	BoundingBoxes       []WMSBoundingBox `json:"boundingBoxes"`
	Visible             *bool            `json:"visible,omitempty"`
	Authority           *Authority       `json:"authority,omitempty"`
	DatasetMetadataURL  *MetadataURL     `json:"datasetMetadataUrl,omitempty"`
	MinScaleDenominator *string          `json:"minscaledenominator,omitempty"`
	MaxScaleDenominator *string          `json:"maxscaledenominator,omitempty"`
	Styles              []Style          `json:"styles"`
	LabelNoClip         bool             `json:"labelNoClip"`
	Data                *Data            `json:"data,omitempty"`
	// Nested structs do not work in crd generation
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Layers *[]Layer `json:"layers,omitempty"`
}

type WMSBoundingBox struct {
	CRS  string            `json:"crs"`
	BBox shared_model.BBox `json:"bbox"`
}

func (wmsBoundingBox *WMSBoundingBox) ToExtent() string {
	return wmsBoundingBox.BBox.ToExtent()
}

func (wmsBoundingBox *WMSBoundingBox) Combine(other *WMSBoundingBox) {
	if wmsBoundingBox.CRS != other.CRS {
		return
	}
	wmsBoundingBox.BBox.Combine(other.BBox)
}

type Authority struct {
	Name                     string `json:"name"`
	URL                      string `json:"url"`
	SpatialDatasetIdentifier string `json:"spatialDatasetIdentifier"`
}

type Style struct {
	Name          string  `json:"name"`
	Title         *string `json:"title"`
	Abstract      *string `json:"abstract"`
	Visualization *string `json:"visualization"`
	Legend        *Legend `json:"legend"`
}

type Legend struct {
	Width   int32  `json:"width"`
	Height  int32  `json:"height"`
	Format  string `json:"format"`
	BlobKey string `json:"blobKey"`
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

func (wmsService *WMSService) GetBoundingBox() WMSBoundingBox {
	var boundingBox *WMSBoundingBox

	allLayers := wmsService.GetAllLayers()
	for _, layer := range allLayers {
		if layer.BoundingBoxes != nil && len(layer.BoundingBoxes) > 0 {
			for _, bbox := range wmsService.Layer.BoundingBoxes {
				if boundingBox == nil {
					boundingBox = &bbox
				} else {
					boundingBox.Combine(&bbox)
				}
			}
		}
	}

	if boundingBox != nil {
		return *boundingBox
	} else {
		return WMSBoundingBox{
			CRS: "EPSG:28992",
			BBox: shared_model.BBox{
				MinX: "-25000",
				MaxX: "280000",
				MinY: "250000",
				MaxY: "860000",
			},
		}
	}

}

func (wmsService *WMSService) GetAllLayers() (layers []Layer) {
	return wmsService.Layer.GetAllLayers()
}

func (layer *Layer) GetAllLayers() (layers []Layer) {
	layers = append(layers, *layer)
	if layer.Layers != nil {
		for _, childLayer := range *layer.Layers {
			layers = append(layers, childLayer.GetAllLayers()...)
		}
	}
	return
}

func (layer *Layer) GetParent(candidateLayer *Layer) *Layer {
	if candidateLayer.Layers == nil {
		return nil
	}

	for _, childLayer := range *candidateLayer.Layers {
		if childLayer.Name == layer.Name {
			return candidateLayer
		} else {
			parent := layer.GetParent(&childLayer)
			if parent != nil {
				return parent
			}
		}
	}
	return nil
}

func (layer *Layer) hasData() bool {
	switch {
	case layer.Data == nil:
		return false
	case layer.Data.Gpkg != nil:
		return true
	case layer.Data.Postgis != nil:
		return true
	case layer.Data.TIF != nil:
		return true
	default:
		return false
	}
}

func (layer *Layer) hasTIFData() bool {
	if !layer.hasData() {
		return false
	}
	return layer.Data.TIF != nil && layer.Data.TIF.BlobKey != ""
}

func (layer *Layer) GetLayerType(service *WMSService) (layerType string) {
	switch {
	case layer.IsDataLayer():
		return DataLayer
	case layer.Name == service.Layer.Name:
		return TopLayer
	default:
		return GroupLayer
	}
}

func (layer *Layer) IsDataLayer() bool {
	return layer.hasData() && (layer.Layers == nil || len(*layer.Layers) == 0)
}

func (layer *Layer) IsGroupLayer() bool {
	return layer.Layers != nil && len(*layer.Layers) > 0 && layer.Visible != nil && *layer.Visible
}

func (layer *Layer) IsTopLayer(service *WMSService) bool {
	return layer.Name == service.Layer.Name
}

func (layer *Layer) hasBoundingBoxForCRS(crs string) bool {
	for _, bbox := range layer.BoundingBoxes {
		if bbox.CRS == crs {
			return true
		}
	}
	return false
}

func (layer *Layer) setInheritedBoundingBoxes() {
	if layer.Layers == nil || len(*layer.Layers) == 0 {
		return
	}

	var updatedLayers []Layer
	for _, childLayer := range *layer.Layers {
		// Inherit parent boundingboxes
		for _, boundingBox := range layer.BoundingBoxes {
			if !childLayer.hasBoundingBoxForCRS(boundingBox.CRS) {
				childLayer.BoundingBoxes = append(childLayer.BoundingBoxes, boundingBox)
			}
		}
		childLayer.setInheritedBoundingBoxes()
		updatedLayers = append(updatedLayers, childLayer)
	}
	*layer.Layers = updatedLayers
}

func (wms *WMS) GetAllLayersWithLegend() (layers []Layer) {
	for _, layer := range wms.Spec.Service.Layer.GetAllLayers() {
		if !layer.hasData() || len(layer.Styles) == 0 {
			continue
		}
		for _, style := range layer.Styles {
			if style.Legend != nil && style.Legend.BlobKey != "" {
				layers = append(layers, layer)
				break
			}
		}
	}
	return
}

func (wms *WMS) GetUniqueTiffBlobKeys() []string {
	blobKeys := map[string]bool{}
	for _, layer := range wms.Spec.Service.Layer.GetAllLayers() {
		if layer.hasTIFData() {
			blobKeys[layer.Data.TIF.BlobKey] = true
		}
	}
	keys := slices.Collect(maps.Keys(blobKeys))
	sort.Strings(keys) // This is only needed for the unit test
	return keys
}

func (wms *WMS) GetAuthority() *Authority {
	if wms.Spec.Service.Layer.Authority != nil {
		return wms.Spec.Service.Layer.Authority
	}

	for _, childLayer := range *wms.Spec.Service.Layer.Layers {
		if childLayer.Authority != nil {
			return childLayer.Authority
		} else if childLayer.Layers != nil {
			for _, grandChildLayer := range *childLayer.Layers {
				if grandChildLayer.Authority != nil {
					return grandChildLayer.Authority
				}
			}
		}
	}

	return nil
}

func (wms *WMS) HasPostgisData() bool {
	for _, layer := range wms.Spec.Service.Layer.GetAllLayers() {
		if layer.Data != nil && layer.Data.Postgis != nil {
			return true
		}
	}
	return false
}

func (wms *WMS) Mapfile() *Mapfile {
	return wms.Spec.Service.Mapfile
}

func (wms *WMS) Type() ServiceType {
	return ServiceTypeWMS
}

func (wms *WMS) PodSpecPatch() *corev1.PodSpec {
	return wms.Spec.PodSpecPatch
}

func (wms *WMS) HorizontalPodAutoscalerPatch() *autoscalingv2.HorizontalPodAutoscalerSpec {
	return wms.Spec.HorizontalPodAutoscalerPatch
}

func (wms *WMS) Options() *Options {
	return &wms.Spec.Options
}

func (wms *WMS) ID() string {
	return Sha1HashOfName(wms)
}

func (wms *WMS) URLPath() string {
	return wms.Spec.Service.URL
}
