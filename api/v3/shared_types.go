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
	URLPath() string
}

type Mapfile struct {
	ConfigMapKeyRef corev1.ConfigMapKeySelector `json:"configMapKeyRef"`
}

// TODO moeten allemaal default waardes hebben
// TODO WMS/WFS splitsen
type Options struct {
	IncludeIngress              bool `json:"includeIngress"`              // default true
	AutomaticCasing             bool `json:"automaticCasing"`             // default true
	ValidateRequests            bool `json:"validateRequests"`            // default true
	RewriteGroupToDataLayers    bool `json:"rewriteGroupToDataLayers"`    // default false
	DisableWebserviceProxy      bool `json:"disableWebserviceProxy"`      // default false
	PrefetchData                bool `json:"prefetchData"`                // default true
	ValidateChildStyleNameEqual bool `json:"validateChildStyleNameEqual"` // default true, moeten eigenlijk weg
}

type Inspire struct {
	ServiceMetadataURL       MetadataURL `json:"serviceMetadataUrl"`
	SpatialDatasetIdentifier *string     `json:"spatialDatasetIdentifier,omitempty"` // TODO required for WFS, doesn't exist in WMS, mogelijk uitsplitsen
	Language                 string      `json:"language"`
}

type MetadataURL struct {
	CSW    *Metadata `json:"csw"`
	Custom *Custom   `json:"custom,omitempty"`
}

type Metadata struct {
	MetadataIdentifier string `json:"metadataIdentifier"`
}

type Custom struct {
	Href string `json:"href"`
	Type string `json:"type"`
}

// minstens 1 verplicht, mogelijk splitsing wms/wfs
type Data struct {
	Gpkg    *Gpkg    `json:"gpkg,omitempty"`
	Postgis *Postgis `json:"postgis,omitempty"`
	TIF     *TIF     `json:"tif,omitempty"` // kan niet voor WFS
}

type Gpkg struct {
	BlobKey      string   `json:"blobKey"`
	TableName    string   `json:"tableName"`
	GeometryType string   `json:"geometryType"`
	Columns      []Column `json:"columns"` // minlenght 1
}

// Postgis - reference to table in a Postgres database
type Postgis struct {
	TableName    string   `json:"tableName"`
	GeometryType string   `json:"geometryType"`
	Columns      []Column `json:"columns"` // minlenght 1
}

type TIF struct {
	BlobKey                     string  `json:"blobKey"`
	Resample                    *string `json:"resample,omitempty"`
	Offsite                     *string `json:"offsite,omitempty"`
	GetFeatureInfoIncludesClass *bool   `json:"getFeatureInfoIncludesClass,omitempty"`
}

type Column struct {
	Name  string  `json:"name"`
	Alias *string `json:"alias,omitempty"`
}

func SetHost(url string) {
	host = strings.TrimSuffix(url, "/")
}

func GetHost() string {
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
