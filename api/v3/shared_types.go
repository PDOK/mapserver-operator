package v3

import (
	corev1 "k8s.io/api/core/v1"
	"net/url"
	"strings"
)

var host string

type Mapfile struct {
	ConfigMapKeyRef corev1.ConfigMapKeySelector `json:"configMapKeyRef"`
}

type Options struct {
	IncludeIngress              bool  `json:"includeIngress"`
	AutomaticCasing             bool  `json:"automaticCasing"`
	ValidateRequests            *bool `json:"validateRequests,omitempty"`
	RewriteGroupToDataLayers    *bool `json:"rewriteGroupToDataLayers,omitempty"`
	DisableWebserviceProxy      *bool `json:"disableWebserviceProxy,omitempty"`
	PrefetchData                *bool `json:"prefetchData,omitempty"`
	ValidateChildStyleNameEqual *bool `json:"validateChildStyleNameEqual,omitempty"`
}

type Inspire struct {
	ServiceMetadataURL       MetadataURL `json:"serviceMetadataUrl"`
	SpatialDatasetIdentifier string      `json:"spatialDatasetIdentifier"`
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

type Data struct {
	Gpkg    *Gpkg    `json:"gpkg,omitempty"`
	Postgis *Postgis `json:"postgis,omitempty"`
	TIF     *TIF     `json:"tif,omitempty"`
}

type Gpkg struct {
	BlobKey      string    `json:"blobKey"`
	TableName    string    `json:"tableName"`
	GeometryType string    `json:"geometryType"`
	Columns      []Columns `json:"columns"`
}

// Postgis - reference to table in a Postgres database
type Postgis struct {
	TableName    string    `json:"tableName"`
	GeometryType string    `json:"geometryType"`
	Columns      []Columns `json:"columns"`
}

type TIF struct {
	BlobKey                     string  `json:"blobKey"`
	Resample                    *string `json:"resample,omitempty"`
	Offsite                     *string `json:"offsite,omitepty"`
	GetFeatureInfoIncludesClass *bool   `json:"getFeatureInfoIncludesClass,omitempty"`
}

type Columns struct {
	Name  string  `json:"name"`
	Alias *string `json:"alias,omitempty"`
}

func SetHost(url string) {
	host = strings.TrimSuffix(url, "/")
}

func GetHost() string {
	return host
}

func GetBaseURLPath[T *WFS | *WMS](o T) string {
	var serviceUrl string
	switch any(o).(type) {
	case *WFS:
		if WFS, ok := any(o).(*WFS); ok {
			serviceUrl = WFS.Spec.Service.URL
		}
	case *WMS:
		if WMS, ok := any(o).(*WMS); ok {
			serviceUrl = WMS.Spec.Service.URL
		}
	}

	parsed, _ := url.Parse(serviceUrl)
	return strings.TrimPrefix(parsed.Path, "/")
}
