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

// WMSWFS is the common interface used for both WMS and WFS resources.
// +kubebuilder:object:generate=false
type WMSWFS interface {
	*WFS | *WMS
	metav1.Object

	Mapfile() *Mapfile
	PodSpecPatch() *corev1.PodSpec
	HorizontalPodAutoscalerPatch() *autoscalingv2.HorizontalPodAutoscalerSpec
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
	ConfigMapKeyRef corev1.ConfigMapKeySelector `json:"configMapKeyRef"`
}

// Options configures optional behaviors of the operator, like ingress, casing, and data prefetching.
// +kubebuilder:validation:Type=object
type Options struct {
	// IncludeIngress dictates whether to deploy an Ingress or ensure none exists.
	// +kubebuilder:default:=true
	IncludeIngress bool `json:"includeIngress"`

	// AutomaticCasing enables automatic conversion from snake_case to camelCase.
	// +kubebuilder:default:=true
	AutomaticCasing bool `json:"automaticCasing"`

	// ValidateRequests enables request validation against the service schema.
	// +kubebuilder:default:=true
	ValidateRequests *bool `json:"validateRequests,omitempty"`

	// RewriteGroupToDataLayers merges group layers into individual data layers.
	// +kubebuilder:default:=false
	RewriteGroupToDataLayers *bool `json:"rewriteGroupToDataLayers,omitempty"`

	// DisableWebserviceProxy disables the built-in proxy for external web services.
	// +kubebuilder:default:=false
	DisableWebserviceProxy *bool `json:"disableWebserviceProxy,omitempty"`

	// Whether to prefetch data from blob storage, and store it on the local filesystem.
	// If `false`, the data will be served directly out of blob storage
	// +kubebuilder:default:=true
	PrefetchData *bool `json:"prefetchData,omitempty"`

	// ValidateChildStyleNameEqual ensures child style names match the parent style.
	// +kubebuilder:default=false
	ValidateChildStyleNameEqual *bool `json:"validateChildStyleNameEqual,omitempty"`
}

// Inspire holds INSPIRE-specific metadata for the service.
// +kubebuilder:validation:Type=object
type Inspire struct {
	// ServiceMetadataURL references the CSW or custom metadata record.
	ServiceMetadataURL MetadataURL `json:"serviceMetadataUrl"`

	// SpatialDatasetIdentifier is the ID uniquely identifying the dataset.
	SpatialDatasetIdentifier string `json:"spatialDatasetIdentifier"`

	// Language of the INSPIRE metadata record (e.g., "nl" or "en").
	Language string `json:"language"`
}

type MetadataURL struct {
	// CSW describes a metadata record via a metadataIdentifier (UUID).
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

// Custom represents a non-CSW metadata link with an href and MIME type.
// +kubebuilder:validation:Schemaless
// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:validation:Type=object
type Custom struct {
	// +kubebuilder:validation:Pattern=`^https?://.*$`
	// +kubebuilder:validation:MinLength=1
	Href string `json:"href"`
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
	// +kubebuilder:validation:Pattern=`\.(tif|tiff)$`
	BlobKey string `json:"blobKey"`

	// Resample method
	Resample *string `json:"resample,omitempty"`

	// Offsite color for nodata removal
	Offsite *string `json:"offsite,omitempty"`

	// Include class names in GetFeatureInfo responses
	GetFeatureInfoIncludesClass *bool `json:"getFeatureInfoIncludesClass,omitempty"`
}

// Column maps a source column name to an optional alias for output.
// +kubebuilder:validation:Type=object
type Column struct {
	// Name of the column in the data source.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Alias for the column in the service output.
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
