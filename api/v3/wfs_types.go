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
	"slices"
	"strings"

	smoothoperatormodel "github.com/pdok/smooth-operator/model"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	Spec   WFSSpec                            `json:"spec"`
	Status smoothoperatormodel.OperatorStatus `json:"status,omitempty"`
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
// +kubebuilder:validation:XValidation:rule="!has(self.ingressRouteUrls) || self.ingressRouteUrls.exists_one(x, x.url == self.service.url)",messageExpression="'ingressRouteUrls should include service.url '+self.service.url"
type WFSSpec struct {
	// Optional lifecycle settings
	Lifecycle *smoothoperatormodel.Lifecycle `json:"lifecycle,omitempty"`

	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// Strategic merge patch for the pod in the deployment. E.g. to patch the resources or add extra env vars.
	PodSpecPatch                 corev1.PodSpec                `json:"podSpecPatch"`
	HorizontalPodAutoscalerPatch *HorizontalPodAutoscalerPatch `json:"horizontalPodAutoscalerPatch,omitempty"`
	// TODO omitting the options field or setting an empty value results in incorrect defaulting of the options
	Options *Options `json:"options,omitempty"`

	// Custom healthcheck options
	HealthCheck *HealthCheckWFS `json:"healthCheck,omitempty"`

	// Optional list of URLs where the service can be reached
	// By default only the spec.service.url is used
	IngressRouteURLs smoothoperatormodel.IngressRouteURLs `json:"ingressRouteUrls,omitempty"`

	// service configuration
	Service WFSService `json:"service"`
}

// +kubebuilder:validation:XValidation:message="otherCrs can't contain the defaultCrs",rule="!has(self.otherCrs) || (has(self.otherCrs) && !(self.defaultCrs in self.otherCrs))",fieldPath=".otherCrs"
type WFSService struct {
	// Geonovum subdomein
	// +kubebuilder:validation:MinLength:=1
	Prefix string `json:"prefix"`

	// URL of the service
	URL smoothoperatormodel.URL `json:"url"`

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
	// +kubebuilder:validation:items:MinLength:=1
	Keywords []string `json:"keywords"`

	// Optional Fees
	// +kubebuilder:validation:MinLength:=1
	Fees *string `json:"fees,omitempty"`

	// AccessConstraints URL
	// +kubebuilder:default="https://creativecommons.org/publicdomain/zero/1.0/deed.nl"
	AccessConstraints smoothoperatormodel.URL `json:"accessConstraints,omitempty"`

	// Default CRS (DataEPSG)
	// +kubebuilder:validation:Pattern:="^EPSG:(28992|25831|25832|3034|3035|3857|4258|4326)$"
	DefaultCrs string `json:"defaultCrs"`

	// Other supported CRS
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:items:Pattern:="^EPSG:(28992|25831|25832|3034|3035|3857|4258|4326)$"
	OtherCrs []string `json:"otherCrs,omitempty"`

	// Service bounding box
	Bbox *Bbox `json:"bbox,omitempty"`

	// CountDefault -> wfs_maxfeatures in mapfile
	// +kubebuilder:validation:Minimum:=1
	CountDefault *int `json:"countDefault,omitempty"`

	// FeatureTypes configurations
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:Type=array
	FeatureTypes []FeatureType `json:"featureTypes"`
}

func (s WFSService) KeywordsIncludingInspireKeyword() []string {
	keywords := s.Keywords
	if s.Inspire != nil && !slices.Contains(keywords, "infoFeatureAccessService") {
		keywords = append(keywords, "infoFeatureAccessService")
	}

	return keywords
}

// HealthCheck is the struct with all fields to configure custom healthchecks
type HealthCheckWFS struct {
	// +kubebuilder:validation:XValidation:rule="self.contains('Service=WFS')",message="a valid healthcheck contains 'Service=WFS'"
	// +kubebuilder:validation:XValidation:rule="self.contains('Request=')",message="a valid healthcheck contains 'Request='"
	Querystring string `json:"querystring"`
	// +kubebuilder:validation:Pattern=(image/png|text/xml|text/html)
	Mimetype string `json:"mimetype"`
}

type Bbox struct {
	// EXTENT/wfs_extent in mapfile
	//nolint:tagliatelle
	// +kubebuilder:validation:Type=object
	DefaultCRS smoothoperatormodel.BBox `json:"defaultCRS"`
}

// FeatureType defines a WFS feature
type FeatureType struct {
	// Name of the feature
	// +kubebuilder:validation:Pattern:=`^\S+$`
	Name string `json:"name"`

	// Title of the feature
	// +kubebuilder:validation:MinLength:=1
	Title string `json:"title"`

	// Abstract of the feature
	// +kubebuilder:validation:MinLength:=1
	Abstract string `json:"abstract"`

	// Keywords of the feature
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:items:MinLength:=1
	Keywords []string `json:"keywords"`

	// Metadata URL
	// +kubebuilder:validation:Type=object
	DatasetMetadataURL *MetadataURL `json:"datasetMetadataUrl,omitempty"`

	// Optional feature bbox
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type:=object
	Bbox *FeatureBbox `json:"bbox,omitempty"`

	// FeatureType data connection
	// +kubebuilder:validation:Type=object
	Data Data `json:"data"`
}

// FeatureBbox is the optional featureType bounding box, if provided it overrides the default extent
type FeatureBbox struct {
	// DefaultCRS defines the EXTENT/wfs_extent for the featureType for use in the mapfile
	//nolint:tagliatelle
	// +kubebuilder:validation:Type=object
	DefaultCRS *smoothoperatormodel.BBox `json:"defaultCRS,omitempty"`

	// WGS84, if provided, gives the same bounding box reprojected into EPSG:4326 for use in the capabilities.
	// +kubebuilder:validation:Type=object
	WGS84 *smoothoperatormodel.BBox `json:"wgs84,omitempty"`
}

func (wfs *WFS) HasPostgisData() bool {
	for _, featureType := range wfs.Spec.Service.FeatureTypes {
		if featureType.Data.Postgis != nil {
			return true
		}
	}
	return false
}

func (wfs *WFS) GroupKind() schema.GroupKind {
	return schema.GroupKind{Group: GroupVersion.Group, Kind: wfs.Kind}
}

func (wfs *WFS) Inspire() *Inspire {
	return wfs.Spec.Service.Inspire
}

func (wfs *WFS) Mapfile() *Mapfile {
	return wfs.Spec.Service.Mapfile
}

func (wfs *WFS) Type() ServiceType {
	return ServiceTypeWFS
}

func (wfs *WFS) TypedName() string {
	name := wfs.GetName()
	typeSuffix := strings.ToLower(string(ServiceTypeWFS))

	if strings.HasSuffix(name, typeSuffix) {
		return name
	}

	return name + "-" + typeSuffix
}

func (wfs *WFS) PodSpecPatch() corev1.PodSpec {
	return wfs.Spec.PodSpecPatch
}

func (wfs *WFS) HorizontalPodAutoscalerPatch() *HorizontalPodAutoscalerPatch {
	return wfs.Spec.HorizontalPodAutoscalerPatch
}

func (wfs *WFS) Options() Options {
	if wfs.Spec.Options == nil {
		return *GetDefaultOptions()
	}

	return *wfs.Spec.Options
}

func (wfs *WFS) URL() smoothoperatormodel.URL {
	return wfs.Spec.Service.URL
}

func (wfs *WFS) DatasetMetadataIDs() []string {
	ids := []string{}

	for _, featureType := range wfs.Spec.Service.FeatureTypes {
		if featureType.DatasetMetadataURL != nil && featureType.DatasetMetadataURL.CSW != nil {
			if id := featureType.DatasetMetadataURL.CSW.MetadataIdentifier; !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}

	return ids
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

func (wfs *WFS) ReadinessQueryString() (string, string, error) {
	if hc := wfs.Spec.HealthCheck; hc != nil {
		return hc.Querystring, hc.Mimetype, nil
	}

	if len(wfs.Spec.Service.FeatureTypes) == 0 {
		return "", "", errors.New("cannot get readiness probe for WFS, featuretypes could not be found")
	}

	return "SERVICE=WFS&VERSION=2.0.0&REQUEST=GetFeature&TYPENAMES=" + wfs.Spec.Service.FeatureTypes[0].Name + "&STARTINDEX=0&COUNT=1", "text/xml", nil
}

func (wfs *WFS) IngressRouteURLs(includeServiceURLWhenEmpty bool) smoothoperatormodel.IngressRouteURLs {
	if len(wfs.Spec.IngressRouteURLs) == 0 {
		if includeServiceURLWhenEmpty {
			return smoothoperatormodel.IngressRouteURLs{{URL: wfs.Spec.Service.URL}}
		}

		return smoothoperatormodel.IngressRouteURLs{}
	}

	return wfs.Spec.IngressRouteURLs
}
