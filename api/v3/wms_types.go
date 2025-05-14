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
	"maps"
	"slices"
	"sort"

	shared_model "github.com/pdok/smooth-operator/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// Optional lifecycle settings
	Lifecycle *shared_model.Lifecycle `json:"lifecycle,omitempty"`

	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// Optional strategic merge patch for the pod in the deployment. E.g. to patch the resources or add extra env vars.
	PodSpecPatch *corev1.PodSpec `json:"podSpecPatch,omitempty"`

	// Optional specification for the HorizontalAutoscaler
	HorizontalPodAutoscalerPatch *HorizontalPodAutoscalerPatch `json:"horizontalPodAutoscalerPatch,omitempty"`

	// Optional options for the configuration of the service.
	// TODO omitting the options field or setting an empty value results in incorrect defaulting of the options
	Options Options `json:"options"`

	// Service specification
	Service WMSService `json:"service"`
}

type WMSService struct {
	// URL of the service
	// +kubebuilder:validation:Format:=uri
	URL string `json:"url"`

	// Title of the service
	// +kubebuilder:validation:MinLength:=1
	Title string `json:"title"`

	// Abstract (short description) of the service
	// +kubebuilder:validation:MinLength:=1
	Abstract string `json:"abstract"`

	// Keywords of the service
	// +kubebuilder:validation:MinItems:=1
	Keywords []string `json:"keywords"`

	// Reference to a CR of Kind OwnerInfo
	// +kubebuilder:validation:MinLength:=1
	OwnerInfoRef string `json:"ownerInfoRef"`

	// AccessConstraints (licence) that are applicable to the service
	// +kubebuilder:validation:Pattern:=`https?://.*`
	// +kubebuilder:default="https://creativecommons.org/publicdomain/zero/1.0/deed.nl"
	AccessConstraints string `json:"accessConstraints,omitempty"`

	// Optional specification Inspire themes and ids
	Inspire *Inspire `json:"inspire,omitempty"`

	// CRS of the data
	// +kubebuilder:validation:Pattern:=`(EPSG|CRS):\d+`
	//nolint:tagliatelle
	DataEPSG string `json:"dataEPSG"`

	// Mapfile setting: Sets the maximum size (in pixels) for both dimensions of the image from a getMap request.
	MaxSize *int32 `json:"maxSize,omitempty"`

	// Mapfile setting: Sets the RESOLUTION field in the mapfile, not used when service.mapfile is configured
	Resolution *int32 `json:"resolution,omitempty"`

	// Mapfile setting: Sets the DEFRESOLUTION field in the mapfile, not used when service.mapfile is configured
	DefResolution *int32 `json:"defResolution,omitempty"`

	// Optional. Required files for the styling of the service
	StylingAssets *StylingAssets `json:"stylingAssets,omitempty"`

	// Custom mapfile
	Mapfile *Mapfile `json:"mapfile,omitempty"`

	// Toplayer
	Layer Layer `json:"layer"`
}

func (s WMSService) KeywordsIncludingInspireKeyword() []string {
	keywords := s.Keywords
	if s.Inspire != nil && !slices.Contains(keywords, "infoMapAccessService") {
		keywords = append(keywords, "infoMapAccessService")
	}

	return keywords
}

// +kubebuilder:validation:XValidation:message="Either blobKeys or configMapRefs is required",rule="has(self.blobKeys) || has(self.configMapRefs)"
type StylingAssets struct {
	// +kubebuilder:validations:MinItems:=1
	BlobKeys []string `json:"blobKeys,omitempty"`

	// +kubebuilder:validations:MinItems:=1
	ConfigMapRefs []ConfigMapRef `json:"configMapRefs,omitempty"`
}

type ConfigMapRef struct {
	// +kubebuilder:validations:MinLength:=1
	Name string `json:"name"`

	// +kubebuilder:validations:MinItems:=1
	Keys []string `json:"keys,omitempty"`
}

// +kubebuilder:validation:XValidation:message="A layer should have sublayers or data, not both", rule="(has(self.data) || has(self.layers)) && !(has(self.data) && has(self.layers))"
// +kubebuilder:validation:XValidation:message="A layer should have keywords when visible", rule="!self.visible || has(self.keywords)"
// +kubebuilder:validation:XValidation:message="A layer should have a title when visible", rule="!self.visible || has(self.title)"
// +kubebuilder:validation:XValidation:message="A layer should have an abstract when visible", rule="!self.visible || has(self.abstract)"
// +kubebuilder:validation:XValidation:message="A layer should have an authority when visible and has a name", rule="!(self.visible && has(self.name)) || has(self.authority)"
// +kubebuilder:validation:XValidation:message="A layer should have a datasetMetadataUrl when visible and has a name", rule="!(self.visible && has(self.name)) || has(self.datasetMetadataUrl)"
type Layer struct {
	// Name of the layer, required for layers on the 2nd or 3rd level
	// +kubebuilder:validations:MinLength:=1
	Name *string `json:"name,omitempty"`

	// Title of the layer
	// +kubebuilder:validations:MinLength:=1
	Title *string `json:"title,omitempty"`

	// Abstract of the layer
	// +kubebuilder:validations:MinLength:=1
	Abstract *string `json:"abstract,omitempty"`

	// Keywords of the layer, required if the layer is visible
	// +kubebuilder:validations:MinItems:=1
	Keywords []string `json:"keywords,omitempty"`

	// BoundingBoxes of the layer. If omitted the boundingboxes of the parent layer of the service is used.
	// +kubebuilder:validations:MinItems:=1
	BoundingBoxes []WMSBoundingBox `json:"boundingBoxes,omitempty"`

	// Whether or not the layer is visible. At least one of the layers must be visible.
	// +kubebuilder:default:=true
	Visible bool `json:"visible"`

	// TODO ??
	Authority *Authority `json:"authority,omitempty"`

	// Links to metadata
	DatasetMetadataURL *MetadataURL `json:"datasetMetadataUrl,omitempty"`

	// The minimum scale at which this layer functions
	// +kubebuilder:validation:Pattern:=`^[0-9]+(.[0-9]+)?$`
	MinScaleDenominator *string `json:"minscaledenominator,omitempty"`

	// The maximum scale at which this layer functions
	// +kubebuilder:validation:Pattern:=`^[1-9][0-9]*(.[0-9]+)?$`
	MaxScaleDenominator *string `json:"maxscaledenominator,omitempty"`

	// List of styles used by the layer
	// +kubebuilder:validations:MinItems:=1
	Styles []Style `json:"styles,omitempty"`

	// Mapfile setting, sets "LABEL_NO_CLIP=ON"
	LabelNoClip bool `json:"labelNoClip,omitempty"`

	// Data (gpkg/postgis/tif) used by the layer
	Data *Data `json:"data,omitempty"`

	// Sublayers of the layer
	// +kubebuilder:validations:MinItems:=1
	Layers []Layer `json:"layers,omitempty"`
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
	// +kubebuilder:validations:MinLength:=1
	Name string `json:"name"`

	// +kubebuilder:validations:MinLength:=1
	Title *string `json:"title,omitempty"`

	// +kubebuilder:validations:MinLength:=1
	Abstract *string `json:"abstract,omitempty"`

	// +kubebuilder:validations:MinLength:=1
	Visualization *string `json:"visualization,omitempty"`

	Legend *Legend `json:"legend,omitempty"`
}

type Legend struct {
	// The width of the legend in px, defaults to 78
	// + kubebuilder:default=78
	Width int32 `json:"width,omitempty"`

	// The height of the legend in px, defaults to 20
	// + kubebuilder:default=20
	Height int32 `json:"height,omitempty"`

	// Format of the legend, defaults to image/png
	// +kubebuilder:default="image/png"
	Format string `json:"format,omitempty"`

	// Location of the legend on the blobstore
	// +kubebuilder:validation:MinLength:=1
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
		if len(layer.BoundingBoxes) > 0 {
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
	}

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

type AnnotatedLayer struct {
	// The name of the group that this layer belongs to, nil if it is not a member of a group. Groups can be a member of the toplayer as a group
	GroupName *string
	// Only for spec.Service.Layer
	IsTopLayer bool
	// Top layer or layer below the toplayer with children itself
	IsGroupLayer bool
	// Contains actual data
	IsDataLayer bool
	Layer       Layer
}

func (wmsService *WMSService) GetAnnotatedLayers() []AnnotatedLayer {
	result := make([]AnnotatedLayer, 0)

	if wmsService.Layer.Name != nil && len(*wmsService.Layer.Name) > 0 {
		firstLayer := AnnotatedLayer{
			GroupName:    nil,
			IsTopLayer:   wmsService.Layer.IsTopLayer(),
			IsGroupLayer: wmsService.Layer.IsGroupLayer(),
			IsDataLayer:  wmsService.Layer.IsDataLayer(),
			Layer:        wmsService.Layer,
		}
		result = append(result, firstLayer)
	}

	for _, subLayer := range wmsService.Layer.Layers {
		groupName := wmsService.Layer.Name
		isGroupLayer := subLayer.IsGroupLayer()
		isDataLayer := !isGroupLayer
		result = append(result, AnnotatedLayer{
			GroupName:    groupName,
			IsTopLayer:   false,
			IsGroupLayer: isGroupLayer,
			IsDataLayer:  isDataLayer,
			Layer:        subLayer,
		})

		for _, subSubLayer := range subLayer.Layers {
			result = append(result, AnnotatedLayer{
				GroupName:    subLayer.Name,
				IsTopLayer:   false,
				IsGroupLayer: false,
				IsDataLayer:  true,
				Layer:        subSubLayer,
			})
		}
	}

	return result
}

func (wmsService *WMSService) GetAllLayers() (layers []Layer) {
	return wmsService.Layer.GetAllLayers()
}

func (layer *Layer) GetAllLayers() (layers []Layer) {
	layers = append(layers, *layer)
	for _, childLayer := range layer.Layers {
		layers = append(layers, childLayer.GetAllLayers()...)
	}
	return
}

func (layer *Layer) GetParent(candidateLayer *Layer) *Layer {
	if candidateLayer.Layers == nil {
		return nil
	}

	for _, childLayer := range candidateLayer.Layers {
		if childLayer.Name == layer.Name {
			return candidateLayer
		}

		parent := layer.GetParent(&childLayer)
		if parent != nil {
			return parent
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
	return layer.hasData() && len(layer.Layers) == 0
}

func (layer *Layer) IsGroupLayer() bool {
	return len(layer.Layers) > 0
}

// IsTopLayer - a layer is a toplayer if and only if it has sublayers that are group layers.
// In other words the layer is level 1 in a 3 level hierarchy.
func (layer *Layer) IsTopLayer() bool {
	if layer.IsGroupLayer() {
		for _, childLayer := range layer.Layers {
			if childLayer.IsGroupLayer() {
				return true
			}
		}
	}

	return false
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
	if len(layer.Layers) == 0 {
		return
	}

	var updatedLayers []Layer
	for _, childLayer := range layer.Layers {
		// Inherit parent boundingboxes
		for _, boundingBox := range layer.BoundingBoxes {
			if !childLayer.hasBoundingBoxForCRS(boundingBox.CRS) {
				childLayer.BoundingBoxes = append(childLayer.BoundingBoxes, boundingBox)
			}
		}
		childLayer.setInheritedBoundingBoxes()
		updatedLayers = append(updatedLayers, childLayer)
	}
	layer.Layers = updatedLayers
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

	for _, childLayer := range wms.Spec.Service.Layer.Layers {
		if childLayer.Authority != nil {
			return childLayer.Authority
		} else if childLayer.Layers != nil {
			for _, grandChildLayer := range childLayer.Layers {
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

func (wms *WMS) HorizontalPodAutoscalerPatch() *HorizontalPodAutoscalerPatch {
	return wms.Spec.HorizontalPodAutoscalerPatch
}

func (wms *WMS) Options() Options {
	return wms.Spec.Options
}

func (wms *WMS) ID() string {
	return Sha1HashOfName(wms)
}

func (wms *WMS) URLPath() string {
	return wms.Spec.Service.URL
}

func (wms *WMS) GeoPackages() []*Gpkg {
	gpkgs := make([]*Gpkg, 0)

	// TODO fix linting (nestif)
	if wms.Spec.Service.Layer.Layers != nil {
		for _, layer := range wms.Spec.Service.Layer.Layers {
			if layer.Data != nil {
				if layer.Data.Gpkg != nil {
					gpkgs = append(gpkgs, layer.Data.Gpkg)
				}
			} else if layer.Layers != nil {
				for _, childLayer := range layer.Layers {
					if childLayer.Data != nil && childLayer.Data.Gpkg != nil {
						gpkgs = append(gpkgs, childLayer.Data.Gpkg)
					}
				}
			}
		}
	}

	return gpkgs
}

//nolint:revive
func (wms *WMS) GetBaseUrl() string {
	return wms.Spec.Service.URL
}
