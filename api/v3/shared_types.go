package v3

import (
	//nolint:gosec
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/url"
	"strings"

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
	MaxReplicas int32                                          `json:"maxReplicas"`
	Metrics     []autoscalingv2.MetricSpec                     `json:"metrics,omitempty"`
	Behavior    *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

// WMSWFS is the common interface used for both WMS and WFS resources.
// +kubebuilder:object:generate=false
type WMSWFS interface {
	*WFS | *WMS
	metav1.Object

	Mapfile() *Mapfile
	PodSpecPatch() *corev1.PodSpec
	HorizontalPodAutoscalerPatch() *HorizontalPodAutoscalerPatch
	Type() ServiceType
	Options() *Options
	HasPostgisData() bool
	// Sha1 hash of the objects name
	ID() string
	// URLPath returns the configured service URL
	URLPath() string

	GeoPackages() []*Gpkg
}

// Mapfile references a ConfigMap key where an external mapfile is stored.
// +kubebuilder:validation:Type=object
type Mapfile struct {
	// +kubebuilder:validation:Type=object
	ConfigMapKeyRef corev1.ConfigMapKeySelector `json:"configMapKeyRef"`
}

// TODO GedefaultOptions
// Options configures optional behaviors of the operator, like ingress, casing, and data prefetching.
// +kubebuilder:validation:Type=object
type Options struct {
	// IncludeIngress dictates whether to deploy an Ingress or ensure none exists.
	// +kubebuilder:default:=true
	IncludeIngress bool `json:"includeIngress,omitempty"`

	// AutomaticCasing enables automatic conversion from snake_case to camelCase.
	// +kubebuilder:default:=true
	AutomaticCasing bool `json:"automaticCasing,omitempty"`

	// ValidateRequests enables request validation against the service schema.
	// +kubebuilder:default:=true
	ValidateRequests bool `json:"validateRequests,omitempty"`

	// RewriteGroupToDataLayers merges group layers into individual data layers.
	// +kubebuilder:default:=false
	RewriteGroupToDataLayers bool `json:"rewriteGroupToDataLayers,omitempty"`

	// DisableWebserviceProxy disables the built-in proxy for external web services.
	// +kubebuilder:default:=false
	DisableWebserviceProxy bool `json:"disableWebserviceProxy,omitempty"`

	// Whether to prefetch data from blob storage, and store it on the local filesystem.
	// If `false`, the data will be served directly out of blob storage
	// +kubebuilder:default:=true
	PrefetchData bool `json:"prefetchData,omitempty"`

	// ValidateChildStyleNameEqual ensures child style names match the parent style.
	// +kubebuilder:default=false
	ValidateChildStyleNameEqual bool `json:"validateChildStyleNameEqual,omitempty"`
}

// Inspire holds INSPIRE-specific metadata for the service.
// +kubebuilder:validation:Type=object
type Inspire struct {
	// ServiceMetadataURL references the CSW or custom metadata record.
	// +kubebuilder:validation:Type=object
	ServiceMetadataURL MetadataURL `json:"serviceMetadataUrl"`

	// SpatialDatasetIdentifier is the ID uniquely identifying the dataset.
	// +kubebuilder:validation:MinLength:=1
	SpatialDatasetIdentifier string `json:"spatialDatasetIdentifier"`

	// Language of the INSPIRE metadata record
	// +kubebuilder:validation:MinLength:=1
	Language string `json:"language"`
}

// TODO one of the two, not both
type MetadataURL struct {
	// CSW describes a metadata record via a metadataIdentifier (UUID) as defined in the OwnerInfo.
	CSW *Metadata `json:"csw"`

	// Custom allows arbitrary href
	Custom *Custom `json:"custom,omitempty"`
}

// Metadata holds the UUID of a CSW metadata record
type Metadata struct {
	// MetadataIdentifier is the record's UUID
	// +kubebuilder:validation:Format:=uuid
	// +kubebuilder:validation:MinLength:=1
	MetadataIdentifier string `json:"metadataIdentifier"`
}

// Custom represents a non-CSW metadata link with a href and MIME type.
// +kubebuilder:validation:Type=object
type Custom struct {
	// +kubebuilder:validation:Pattern=`^https?://.*$`
	// +kubebuilder:validation:MinLength=1
	Href string `json:"href"`

	// MIME type of the custom link
	// +kubebuilder:validation:MinLength=1
	Type string `json:"type"`
}

// Data holds the data source configuration
// +kubebuilder:validation:XValidation:rule="has(self.gpkg) || has(self.tif) || has(self.postgis)", message="Atleast one of the datasource should be provided (postgis, gpkg, tif)"
type Data struct {
	// Gpkg configures a GeoPackage file source
	Gpkg *Gpkg `json:"gpkg,omitempty"`

	// Postgis configures a Postgis table source
	Postgis *Postgis `json:"postgis,omitempty"`

	// TIF configures a GeoTIF raster source
	TIF *TIF `json:"tif,omitempty"`
}

// Gpkg configures a Geopackage data source
// +kubebuilder:validation:Type=object
type Gpkg struct {
	// Blobkey identifies the location/bucket of the .gpkg file
	// +kubebuilder:validation:Pattern=`\.gpkg$`
	// +kubebuilder:validation:MinLength:=1
	BlobKey string `json:"blobKey"`

	// TableName is the table within the geopackage
	// +kubebuilder:validation:MinLength:=1
	TableName string `json:"tableName"`

	// GeometryType of the table, must match an OGC type
	// +kubebuilder:validation:Pattern:=`^(Multi)?(Point|LineString|Polygon)$`
	// +kubebuilder:validation:MinLength:=1
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
	// +kubebuilder:validation:MinLength:=1
	GeometryType string `json:"geometryType"`

	// Columns to expose from table
	// +kubebuilder:validation:MinItems=1
	Columns []Column `json:"columns"`
}

// TIF configures a GeoTIFF raster data source
// +kubebuilder:validation:Type=object
type TIF struct {
	// BlobKey to the TIFF file
	// +kubebuilder:validation:Pattern=`\.(tif?f|vrt)$`
	// +kubebuilder:validation:MinLength:=1
	BlobKey string `json:"blobKey"`

	// TODO pattern: "(NEAREST|AVERAGE|BILINEAR)"
	// "This option can be used to control the resampling kernel used sampling raster images, optional"
	// +kubebuilder:validation:MinLength:=1
	Resample *string `json:"resample,omitempty"`

	// TODO pattern: '(#[0-9A-F]{2}[0-9A-F]{2}[0-9A-F]{2}([0-9A-F]{2})?)|([0-9]{1,3}\s[0-9]{1,3}\s[0-9]{1,3})'
	// "Sets the color index to treat as transparent for raster layers, optional"
	// +kubebuilder:validation:MinLength:=1
	Offsite *string `json:"offsite,omitempty"`

	// TODO default false
	// "When a band represents nominal or ordinal data the class name (from styling) can be included in the getFeatureInfo"
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
	if includeProtocol {
		return host
	} else if strings.HasPrefix(host, "http") {
		return strings.Split(host, "://")[1]
	}

	return host
}

func GetBaseURLPath[T WMSWFS](o T) string {
	serviceURL := o.URLPath()
	parsed, _ := url.Parse(serviceURL)
	return strings.TrimPrefix(parsed.Path, "/")
}

func (d *Data) GetColumns() *[]Column {
	switch {
	case d.Gpkg != nil:
		return &d.Gpkg.Columns
	case d.Postgis != nil:
		return &d.Postgis.Columns
	default:
		return nil
	}
}

func (d *Data) GetTableName() *string {
	switch {
	case d.Gpkg != nil:
		return &d.Gpkg.TableName
	case d.Postgis != nil:
		return &d.Postgis.TableName
	default:
		return nil
	}
}

func (d *Data) GetGeometryType() *string {
	switch {
	case d.Gpkg != nil:
		return &d.Gpkg.GeometryType
	case d.Postgis != nil:
		return &d.Postgis.GeometryType
	default:
		return nil
	}
}

func Sha1HashOfName[O WMSWFS](obj O) string {
	//nolint:gosec
	s := sha1.New()
	_, _ = io.WriteString(s, obj.GetName())

	return hex.EncodeToString(s.Sum(nil))
}

func (o *Options) UseWebserviceProxy() bool {
	// options.DisableWebserviceProxy not set or false
	return o != nil && (o.DisableWebserviceProxy == nil || !*o.DisableWebserviceProxy)
}
