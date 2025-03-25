package v3

import (
	corev1 "k8s.io/api/core/v1"
	"strings"
)

var baseURL string

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

func SetBaseURL(url string) {
	baseURL = strings.TrimSuffix(url, "/")
}

func GetBaseURL() string {
	return baseURL
}
