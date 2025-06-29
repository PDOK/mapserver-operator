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
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	smoothoperatormodel "github.com/pdok/smooth-operator/model"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	TopLayer   = "topLayer"
	DataLayer  = "dataLayer"
	GroupLayer = "groupLayer"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:subresource:status
// versionName=v3
// +kubebuilder:resource:path=wms
// +kubebuilder:resource:categories=pdok
// +kubebuilder:printcolumn:name="ReadyPods",type=integer,JSONPath=`.status.podSummary[0].ready`
// +kubebuilder:printcolumn:name="DesiredPods",type=integer,JSONPath=`.status.podSummary[0].total`
// +kubebuilder:printcolumn:name="ReconcileStatus",type=string,JSONPath=`.status.conditions[?(@.type == "Reconciled")].reason`

// WMS is the Schema for the wms API.
type WMS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WMSSpec                            `json:"spec"`
	Status smoothoperatormodel.OperatorStatus `json:"status,omitempty"`
}

func (wms *WMS) OperatorStatus() *smoothoperatormodel.OperatorStatus {
	return &wms.Status
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

// WMSSpec defines the desired state of WMS.
// +kubebuilder:validation:XValidation:rule="!has(self.ingressRouteUrls) || self.ingressRouteUrls.exists_one(x, x.url == self.service.url)",messageExpression="'ingressRouteUrls should include service.url '+self.service.url"
type WMSSpec struct {
	// Optional lifecycle settings
	Lifecycle *smoothoperatormodel.Lifecycle `json:"lifecycle,omitempty"`

	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// Strategic merge patch for the pod in the deployment. E.g. to patch the resources or add extra env vars.
	PodSpecPatch corev1.PodSpec `json:"podSpecPatch"`

	// Optional specification for the HorizontalAutoscaler
	HorizontalPodAutoscalerPatch *HorizontalPodAutoscalerPatch `json:"horizontalPodAutoscalerPatch,omitempty"`

	// Optional options for the configuration of the service.
	// TODO omitting the options field or setting an empty value results in incorrect defaulting of the options
	Options *Options `json:"options,omitempty"`

	// Custom healthcheck options
	HealthCheck *HealthCheckWMS `json:"healthCheck,omitempty"`

	// Optional list of URLs where the service can be reached
	// By default only the spec.service.url is used
	IngressRouteURLs smoothoperatormodel.IngressRouteURLs `json:"ingressRouteUrls,omitempty"`

	// Service specification
	Service WMSService `json:"service"`
}

// +kubebuilder:validation:XValidation:message="service requires styling, either through service.mapfile, or stylingAssets.configMapRefs",rule=has(self.mapfile) || (has(self.stylingAssets) && has(self.stylingAssets.configMapRefs))
// +kubebuilder:validation:XValidation:message="when using service.mapfile, don't include stylingAssets.configMapRefs",rule=!has(self.mapfile) || (!has(self.stylingAssets) || !has(self.stylingAssets.configMapRefs))
type WMSService struct {
	BaseService `json:",inline"`

	// Config for Inspire services
	Inspire *Inspire `json:"inspire,omitempty"`

	// CRS of the data
	// +kubebuilder:validation:Pattern:=`(EPSG|CRS):\d+`
	//nolint:tagliatelle
	DataEPSG string `json:"dataEPSG"`

	// Mapfile setting: Sets the maximum size (in pixels) for both dimensions of the image from a getMap request.
	// +kubebuilder:validation:Minimum:=1
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

func (wmsService WMSService) KeywordsIncludingInspireKeyword() []string {
	keywords := wmsService.Keywords
	if wmsService.Inspire != nil && !slices.Contains(keywords, "infoMapAccessService") {
		keywords = append(keywords, "infoMapAccessService")
	}

	return keywords
}

// HealthCheck is the struct with all fields to configure custom healthchecks
// +kubebuilder:validation:XValidation:rule="!has(self.querystring) || has(self.mimetype)",message="mimetype is required when a querystring is used"
// +kubebuilder:validation:XValidation:rule="(has(self.boundingbox) || has(self.querystring)) && !(has(self.querystring) && has(self.boundingbox))", message="healthcheck should have exactly 1 of querystring + mimetype or boundingbox"
// +kubebuilder:validation:XValidation:rule="(has(self.boundingbox) || has(self.mimetype)) && !(has(self.mimetype) && has(self.boundingbox))", message="healthcheck should have exactly 1 of querystring + mimetype or boundingbox"
type HealthCheckWMS struct {
	// +kubebuilder:validation:XValidation:rule="self.lowerAscii().contains('service=wms')",message="a valid healthcheck contains 'SERVICE=WMS'"
	// +kubebuilder:validation:XValidation:rule="self.lowerAscii().contains('request=')",message="a valid healthcheck contains 'REQUEST='"
	Querystring *string `json:"querystring,omitempty"`
	// +kubebuilder:validation:Pattern=(image/png|text/xml|text/html)
	Mimetype *string `json:"mimetype,omitempty"`

	Boundingbox *smoothoperatormodel.BBox `json:"boundingbox,omitempty"`
}

// StylingAssets contains the files references needed for styling
// +kubebuilder:validation:XValidation:message="At least one of blobKeys or configMapRefs is required",rule="has(self.blobKeys) || has(self.configMapRefs)"
type StylingAssets struct {
	// BlobKeys contains symbol image (.png/.svg) or font (.ttf) keys on blob storage, format: container/key/file.(png|ttf)
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:items:Pattern:=^.+\/.+\/.+\.(png|ttf|svg)$
	BlobKeys []string `json:"blobKeys,omitempty"`

	// +kubebuilder:validation:MinItems:=1
	ConfigMapRefs []ConfigMapRef `json:"configMapRefs,omitempty"`
}

type ConfigMapRef struct {
	// Name is the name of the ConfigMap
	// +kubebuilder:validation:MinLength:=1
	Name string `json:"name"`

	// Keys contains styling assets that contain mapfile code (.style|.symbol), required if you use symbols in your styles
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:items:Pattern:=^\S*.\.(style|symbol)
	Keys []string `json:"keys,omitempty"`
}

// +kubebuilder:validation:XValidation:message="A layer should have exactly one of sublayers or data", rule="(has(self.data) || has(self.layers)) && !(has(self.data) && has(self.layers))"
// +kubebuilder:validation:XValidation:message="A layer with data attribute should have styling", rule="!has(self.data) || has(self.styles)"
// +kubebuilder:validation:XValidation:message="A layer should have a title when visible", rule="!self.visible || has(self.title)"
// +kubebuilder:validation:XValidation:message="A layer should have an abstract when visible", rule="!self.visible || has(self.abstract)"
// +kubebuilder:validation:XValidation:message="A layer should have keywords when visible", rule="!self.visible || has(self.keywords)"
type Layer struct {
	// Name of the layer, required for layers on the 2nd or 3rd level
	// +kubebuilder:validation:MinLength:=1
	Name *string `json:"name,omitempty"`

	// Title of the layer
	// +kubebuilder:validation:MinLength:=1
	Title *string `json:"title,omitempty"`

	// Abstract of the layer
	// +kubebuilder:validation:MinLength:=1
	Abstract *string `json:"abstract,omitempty"`

	// Keywords of the layer, required if the layer is visible
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:items:MinLength:=1
	Keywords []string `json:"keywords,omitempty"`

	// BoundingBoxes of the layer. If omitted the boundingboxes of the parent layer of the service is used.
	// +kubebuilder:validation:MinItems:=1
	BoundingBoxes []WMSBoundingBox `json:"boundingBoxes,omitempty"`

	// Whether or not the layer is visible. At least one of the layers must be visible.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
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
	// +kubebuilder:validation:MinItems:=1
	Styles []Style `json:"styles,omitempty"`

	// Mapfile setting, sets "LABEL_NO_CLIP=ON"
	LabelNoClip bool `json:"labelNoClip,omitempty"`

	// Data (gpkg/postgis/tif) used by the layer
	Data *Data `json:"data,omitempty"`

	// Sublayers of the layer
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:Type=array
	Layers []Layer `json:"layers,omitempty"`
}

type WMSBoundingBox struct {
	// +kubebuilder:validation:Pattern:="^(EPSG:(28992|25831|25832|3034|3035|3857|4258|4326)|CRS:84)$"
	CRS  string                   `json:"crs"`
	BBox smoothoperatormodel.BBox `json:"bbox"`
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
	// +kubebuilder:validation:MinLength:=1
	Name string `json:"name"`

	// +kubebuilder:validation:MinLength:=1
	Title *string `json:"title,omitempty"`

	// +kubebuilder:validation:MinLength:=1
	Abstract *string `json:"abstract,omitempty"`

	// +kubebuilder:validation:MinLength:=1
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

// WMSOptions are the Options exclusively used by the WMS
// +kubebuilder:validation:Type=object
type WMSOptions struct {

	// ValidateRequests enables request validation against the service schema.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	ValidateRequests bool `json:"validateRequests"`

	// RewriteGroupToDataLayers merges group layers into individual data layers.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	RewriteGroupToDataLayers bool `json:"rewriteGroupToDataLayers"`

	// DisableWebserviceProxy disables the built-in proxy for external web services.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	DisableWebserviceProxy bool `json:"disableWebserviceProxy"`

	// ValidateChildStyleNameEqual ensures child style names match the parent style.
	// +kubebuilder:default=false
	// +kubebuilder:validation:Optional
	ValidateChildStyleNameEqual bool `json:"validateChildStyleNameEqual"`
}

func (wmsService *WMSService) GetBoundingBox() WMSBoundingBox {
	var boundingBox *WMSBoundingBox

	allLayers := wmsService.GetAnnotatedLayers()
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
		BBox: smoothoperatormodel.BBox{
			MinX: "-25000",
			MaxX: "280000",
			MinY: "250000",
			MaxY: "860000",
		},
	}
}

func (stylingAssets *StylingAssets) GetAllConfigMapRefKeys() []string {
	keys := []string{}
	if stylingAssets != nil {
		for _, cmRef := range stylingAssets.ConfigMapRefs {
			keys = append(keys, cmRef.Keys...)
		}
	}
	return keys
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
	Layer
}

func (wmsService *WMSService) GetAnnotatedLayers() []AnnotatedLayer {
	result := make([]AnnotatedLayer, 0)

	result = append(result, AnnotatedLayer{
		GroupName:    nil,
		IsTopLayer:   true,
		IsGroupLayer: true,
		IsDataLayer:  false,
		Layer:        wmsService.Layer,
	})

	for _, middleLayer := range wmsService.Layer.Layers {
		result = append(result, AnnotatedLayer{
			GroupName:    wmsService.Layer.Name,
			IsTopLayer:   false,
			IsGroupLayer: middleLayer.IsGroupLayer(),
			IsDataLayer:  middleLayer.IsDataLayer(),
			Layer:        middleLayer,
		})

		for _, bottomLayer := range middleLayer.Layers {
			result = append(result, AnnotatedLayer{
				GroupName:    middleLayer.Name,
				IsTopLayer:   false,
				IsGroupLayer: false,
				IsDataLayer:  true,
				Layer:        bottomLayer,
			})
		}
	}

	return result
}

// GetAllSublayers - get all sublayers of a layer, the result does not include the layer itself
func (layer *Layer) GetAllSublayers() []Layer {
	layers := layer.Layers
	for _, childLayer := range layer.Layers {
		layers = append(layers, childLayer.GetAllSublayers()...)
	}
	return layers
}

func (wmsService *WMSService) GetParentLayer(layer Layer) *Layer {
	if wmsService.Layer.Layers == nil {
		return nil
	}

	for _, middleLayer := range wmsService.Layer.Layers {
		if middleLayer.Name == layer.Name {
			return &wmsService.Layer
		}

		for _, bottomLayer := range middleLayer.Layers {
			if bottomLayer.Name == layer.Name {
				return &middleLayer
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

func (wms *WMS) GetAllLayersWithLegend() (layers []AnnotatedLayer) {
	for _, layer := range wms.Spec.Service.GetAnnotatedLayers() {
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
	for _, layer := range wms.Spec.Service.GetAnnotatedLayers() {
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
	for _, layer := range wms.Spec.Service.GetAnnotatedLayers() {
		if layer.Data != nil && layer.Data.Postgis != nil {
			return true
		}
	}
	return false
}

func (wms *WMS) GroupKind() schema.GroupKind {
	return schema.GroupKind{Group: GroupVersion.Group, Kind: wms.Kind}
}

func (wms *WMS) Inspire() *WFSInspire {
	if wms.Spec.Service.Inspire != nil {
		return &WFSInspire{Inspire: *wms.Spec.Service.Inspire}
	}
	return nil
}

func (wms *WMS) Mapfile() *Mapfile {
	return wms.Spec.Service.Mapfile
}

func (wms *WMS) Type() ServiceType {
	return ServiceTypeWMS
}

func (wms *WMS) TypedName() string {
	name := wms.GetName()
	typeSuffix := strings.ToLower(string(ServiceTypeWMS))

	if strings.HasSuffix(name, typeSuffix) {
		return name
	}

	return name + "-" + typeSuffix
}

func (wms *WMS) PodSpecPatch() corev1.PodSpec {
	return wms.Spec.PodSpecPatch
}

func (wms *WMS) HorizontalPodAutoscalerPatch() *HorizontalPodAutoscalerPatch {
	return wms.Spec.HorizontalPodAutoscalerPatch
}

func (wms *WMS) Options() Options {
	if wms.Spec.Options == nil {
		return *GetDefaultOptions()
	}

	return *wms.Spec.Options
}

func (wms *WMS) URL() smoothoperatormodel.URL {
	return wms.Spec.Service.URL
}

func (wms *WMS) DatasetMetadataIDs() []string {
	ids := []string{}

	for _, layer := range wms.Spec.Service.GetAnnotatedLayers() {
		if layer.DatasetMetadataURL != nil && layer.DatasetMetadataURL.CSW != nil {
			if id := layer.DatasetMetadataURL.CSW.MetadataIdentifier; !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}

	return ids
}

func (wms *WMS) GeoPackages() []*Gpkg {
	gpkgs := make([]*Gpkg, 0)

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

	return gpkgs
}

func (wms *WMS) HealthCheckBBox() string {
	if hc := wms.Spec.HealthCheck; hc != nil && hc.Boundingbox != nil {
		return strings.ReplaceAll(hc.Boundingbox.ToExtent(), " ", ",")
	}

	return "190061.4619730016857,462435.5987861062749,202917.7508707302331,473761.6884966178914"
}

func (wms *WMS) ReadinessQueryString() (string, string, error) {
	if hc := wms.Spec.HealthCheck; hc != nil && hc.Querystring != nil {
		return *hc.Querystring, *hc.Mimetype, nil
	}

	firstDataLayerName := ""
	for _, layer := range wms.Spec.Service.GetAnnotatedLayers() {
		if layer.IsDataLayer {
			firstDataLayerName = *layer.Name
			break
		}
	}
	if firstDataLayerName == "" {
		return "", "", errors.New("cannot get readiness probe for WMS, the first datalayer could not be found")
	}

	return fmt.Sprintf("SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&BBOX=%s&CRS=EPSG:28992&WIDTH=100&HEIGHT=100&LAYERS=%s&STYLES=&FORMAT=image/png", wms.HealthCheckBBox(), firstDataLayerName), "image/png", nil
}

func (wms *WMS) IngressRouteURLs(includeServiceURLWhenEmpty bool) smoothoperatormodel.IngressRouteURLs {
	if len(wms.Spec.IngressRouteURLs) == 0 {
		if includeServiceURLWhenEmpty {
			return smoothoperatormodel.IngressRouteURLs{{URL: wms.Spec.Service.URL}}
		}

		return smoothoperatormodel.IngressRouteURLs{}
	}

	return wms.Spec.IngressRouteURLs
}

func (wms *WMS) OwnerInfoRef() string {
	return wms.Spec.Service.OwnerInfoRef
}
