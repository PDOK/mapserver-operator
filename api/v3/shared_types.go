package v3

import (
	"strings"

	smoothoperatormodel "github.com/pdok/smooth-operator/model"

	"k8s.io/apimachinery/pkg/runtime/schema"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var host string

type ServiceType string

const (
	ServiceTypeWMS ServiceType = "WMS"
	ServiceTypeWFS ServiceType = "WFS"
)

// HorizontalPodAutoscalerPatch - copy of autoscalingv2.HorizontalPodAutoscalerSpec without ScaleTargetRef
// This way we don't have to specify the scaleTargetRef field in the CRD.
type HorizontalPodAutoscalerPatch struct {
	MinReplicas *int32                                         `json:"minReplicas,omitempty"`
	MaxReplicas *int32                                         `json:"maxReplicas,omitempty"`
	Metrics     []autoscalingv2.MetricSpec                     `json:"metrics,omitempty"`
	Behavior    *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

// WMSWFS is the common interface used for both WMS and WFS resources.
// +kubebuilder:object:generate=false
type WMSWFS interface {
	*WFS | *WMS
	metav1.Object

	GroupKind() schema.GroupKind
	Inspire() *WFSInspire
	Mapfile() *Mapfile
	PodSpecPatch() corev1.PodSpec
	HorizontalPodAutoscalerPatch() *HorizontalPodAutoscalerPatch
	Type() ServiceType
	TypedName() string
	Options() Options
	HasPostgisData() bool
	OwnerInfoRef() string

	// URL returns the configured service URL
	URL() smoothoperatormodel.URL
	IngressRouteURLs(includeServiceURLWhenEmpty bool) smoothoperatormodel.IngressRouteURLs

	// DatasetMetadataIds returns list of all configured metadata identifiers configured on Layers or Featuretypes
	DatasetMetadataIDs() []string

	GeoPackages() []*Gpkg

	ReadinessQueryString() (string, string, error)
}

// Mapfile references a ConfigMap key where an external mapfile is stored.
// +kubebuilder:validation:Type=object
type Mapfile struct {
	// +kubebuilder:validation:Type=object
	ConfigMapKeyRef corev1.ConfigMapKeySelector `json:"configMapKeyRef"`
}

// BaseOptions for all apis
type BaseOptions struct {
	// IncludeIngress dictates whether to deploy an Ingress or ensure none exists.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	IncludeIngress bool `json:"includeIngress"`

	// AutomaticCasing enables automatic conversion from snake_case to camelCase.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	AutomaticCasing bool `json:"automaticCasing"`

	// Whether to prefetch data from blob storage, and store it on the local filesystem.
	// If `false`, the data will be served directly out of blob storage
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	PrefetchData bool `json:"prefetchData"`
}

// Options configures optional behaviors of the operator, like ingress, casing, and data prefetching.
// +kubebuilder:validation:Type=object
type Options struct {
	BaseOptions `json:",inline"`
	WMSOptions  `json:",inline"`
}

func GetDefaultOptions() *Options {
	return &Options{
		BaseOptions: BaseOptions{
			IncludeIngress:  true,
			AutomaticCasing: true,
			PrefetchData:    true,
		},
		WMSOptions: WMSOptions{
			ValidateRequests:            true,
			RewriteGroupToDataLayers:    false,
			DisableWebserviceProxy:      false,
			ValidateChildStyleNameEqual: false,
		},
	}
}

// BaseService holds all shared Services field for all apis
type BaseService struct {
	// Geonovum subdomein
	// +kubebuilder:validation:MinLength:=1
	Prefix string `json:"prefix"`

	// URL of the service
	URL smoothoperatormodel.URL `json:"url"`

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
}

// Inspire holds INSPIRE-specific metadata for the service.
// +kubebuilder:validation:Type=object
type Inspire struct {
	// ServiceMetadataURL references the CSW or custom metadata record.
	// +kubebuilder:validation:Type=object
	ServiceMetadataURL MetadataURL `json:"serviceMetadataUrl"`

	// Language of the INSPIRE metadata record
	// +kubebuilder:validation:Pattern:=`bul|cze|dan|dut|eng|est|fin|fre|ger|gre|hun|gle|ita|lav|lit|mlt|pol|por|rum|slo|slv|spa|swe`
	Language string `json:"language"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.csw) || has(self.custom)) && !(has(self.csw) && has(self.custom))", message="metadataUrl should have exactly 1 of csw or custom"
type MetadataURL struct {
	// CSW describes a metadata record via a metadataIdentifier (UUID) as defined in the OwnerInfo.
	CSW *Metadata `json:"csw,omitempty"`

	// Custom allows arbitrary href
	Custom *Custom `json:"custom,omitempty"`
}

// Metadata holds the UUID of a CSW metadata record
type Metadata struct {
	// MetadataIdentifier is the record's UUID
	// +kubebuilder:validation:Pattern:=`^[0-9a-zA-Z]{8}\-[0-9a-zA-Z]{4}\-[0-9a-zA-Z]{4}\-[0-9a-zA-Z]{4}\-[0-9a-zA-Z]{12}$`
	MetadataIdentifier string `json:"metadataIdentifier"`
}

// Custom represents a non-CSW metadata link with a href and MIME type.
// +kubebuilder:validation:Type=object
type Custom struct {
	// Href of the custom metadata url
	Href smoothoperatormodel.URL `json:"href"`

	// MIME type of the custom link
	// +kubebuilder:validation:MinLength=1
	Type string `json:"type"`
}

// BaseData holds the data source configuration for gpkg and postgis
type BaseData struct {
	// Gpkg configures a GeoPackage file source
	Gpkg *Gpkg `json:"gpkg,omitempty"`

	// Postgis configures a Postgis table source
	Postgis *Postgis `json:"postgis,omitempty"`
}

// Data holds the data source configuration
// +kubebuilder:validation:XValidation:rule="has(self.gpkg) || has(self.tif) || has(self.postgis)", message="Atleast one of the datasource should be provided (postgis, gpkg, tif)"
type Data struct {
	BaseData `json:",inline"`

	// TIF configures a GeoTIF raster source
	TIF *TIF `json:"tif,omitempty"`
}

// Gpkg configures a Geopackage data source
// +kubebuilder:validation:Type=object
type Gpkg struct {
	// Blobkey identifies the location/bucket of the .gpkg file
	// +kubebuilder:validation:Pattern:=^.+\/.+\/.+\.gpkg$
	BlobKey string `json:"blobKey"`

	// TableName is the table within the geopackage
	// +kubebuilder:validation:MinLength:=1
	TableName string `json:"tableName"`

	// GeometryType of the table, must match an OGC type
	// +kubebuilder:validation:Pattern:=`^(Multi)?(Point|LineString|Polygon)$`
	GeometryType string `json:"geometryType"`

	// Columns to visualize for this table
	// +kubebuilder:validation:MinItems:=1
	Columns []Column `json:"columns"`
}

// Postgis - reference to table in a Postgres database
// +kubebuilder:validation:Type=object
type Postgis struct {
	// TableName in postGIS
	// +kubebuilder:validation:MinLength=1
	TableName string `json:"tableName"`

	// GeometryType of the table
	// +kubebuilder:validation:Pattern=`^(Multi)?(Point|LineString|Polygon)$`
	GeometryType string `json:"geometryType"`

	// Columns to expose from table
	// +kubebuilder:validation:MinItems=1
	Columns []Column `json:"columns"`
}

// TIF configures a GeoTIFF raster data source
// +kubebuilder:validation:Type=object
type TIF struct {
	// BlobKey to the TIFF file
	// +kubebuilder:validation:Pattern:=`^.+\/.+\/.+\.(tif?f|vrt)$`
	BlobKey string `json:"blobKey"`

	// This option can be used to control the resampling kernel used sampling raster images, optional
	// +kubebuilder:validation:Pattern=`(NEAREST|AVERAGE|BILINEAR)`
	// +kubebuilder:default=NEAREST
	Resample string `json:"resample,omitempty"`

	// Controls the smoothing of the image on a certain point. Bigger value gives a smoother/better picture but
	// results in slower web responses, optional
	// +kubebuilder:validation:Pattern="^-?[0-9]+([.][0-9]*)?$"
	// +kubebuilder:default=2.5
	OversampleRatio string `json:"oversampleRatio,omitempty"`

	// Sets the color index to treat as transparent for raster layers, optional, hex or rgb
	// +kubebuilder:validation:Pattern=`(#[0-9A-F]{6}([0-9A-F]{2})?)|([0-9]{1,3}\s[0-9]{1,3}\s[0-9]{1,3})`
	Offsite *string `json:"offsite,omitempty"`

	// "When a band represents nominal or ordinal data the class name (from styling) can be included in the getFeatureInfo"
	// +kubebuilder:default:=false
	GetFeatureInfoIncludesClass bool `json:"getFeatureInfoIncludesClass,omitempty"`
}

// Column maps a source column name to an optional alias for output.
// +kubebuilder:validation:Type=object
type Column struct {
	// Name of the column in the data source.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Alias for the column in the service output.
	// +kubebuilder:validation:MinLength=1
	Alias *string `json:"alias,omitempty"`
}

func SetHost(url string) {
	host = strings.TrimSuffix(url, "/")
}

func GetHost(includeProtocol bool) string {
	if !includeProtocol && strings.HasPrefix(host, "http") {
		return strings.Split(host, "://")[1]
	}

	return host
}

func (d *BaseData) GetColumns() *[]Column {
	switch {
	case d.Gpkg != nil:
		return &d.Gpkg.Columns
	case d.Postgis != nil:
		return &d.Postgis.Columns
	default:
		return nil
	}
}

func (d *BaseData) GetTableName() *string {
	switch {
	case d.Gpkg != nil:
		return &d.Gpkg.TableName
	case d.Postgis != nil:
		return &d.Postgis.TableName
	default:
		return nil
	}
}

func (d *BaseData) GetGeometryType() *string {
	switch {
	case d.Gpkg != nil:
		return &d.Gpkg.GeometryType
	case d.Postgis != nil:
		return &d.Postgis.GeometryType
	default:
		return nil
	}
}

func (o Options) UseWebserviceProxy() bool {
	// options.DisableWebserviceProxy not set or false
	return !o.DisableWebserviceProxy
}
