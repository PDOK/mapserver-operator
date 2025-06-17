package mapfilegenerator

import (
	"path"
	"regexp"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

//nolint:tagliatelle
type BaseServiceInput struct {
	Title             string   `json:"service_title"`
	Abstract          string   `json:"service_abstract"`
	Keywords          string   `json:"service_keywords"`
	Extent            string   `json:"service_extent"`
	NamespacePrefix   string   `json:"service_namespace_prefix"`
	NamespaceURI      string   `json:"service_namespace_uri"`
	OnlineResource    string   `json:"service_onlineresource"`
	Path              string   `json:"service_path"`
	MetadataID        string   `json:"service_metadata_id"`
	DatasetOwner      *string  `json:"dataset_owner,omitempty"`
	AuthorityURL      *string  `json:"authority_url,omitempty"`
	AutomaticCasing   bool     `json:"automatic_casing"`
	DataEPSG          string   `json:"data_epsg"`
	EPSGList          []string `json:"epsg_list"`
	DebugLevel        int      `json:"service_debug_level,omitempty"`
	AccessConstraints string   `json:"service_accessconstraints"`
}

//nolint:tagliatelle
type WFSInput struct {
	BaseServiceInput
	MaxFeatures string     `json:"service_wfs_maxfeatures"`
	Layers      []WFSLayer `json:"layers"`
}

//nolint:tagliatelle
type WMSInput struct {
	BaseServiceInput
	Layers          []WMSLayer   `json:"layers"`
	GroupLayers     []GroupLayer `json:"group_layers"`
	Symbols         []string     `json:"symbols"`
	Fonts           *string      `json:"fonts,omitempty"`
	Templates       string       `json:"templates,omitempty"`
	OutputFormatJpg string       `json:"outputformat_jpg"`
	OutputFormatPng string       `json:"outputformat_png8"`
	MaxSize         string       `json:"maxSize"`
	TopLevelName    string       `json:"top_level_name,omitempty"`
	Resolution      string       `json:"resolution,omitempty"`
	DefResolution   string       `json:"defresolution,omitempty"`
}

//nolint:tagliatelle
type BaseLayer struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Abstract       string   `json:"abstract"`
	Keywords       string   `json:"keywords"`
	Extent         string   `json:"layer_extent"`
	MetadataID     string   `json:"dataset_metadata_id"`
	Columns        []Column `json:"columns,omitempty"`
	GeometryType   *string  `json:"geometry_type,omitempty"`
	GeopackagePath *string  `json:"gpkg_path,omitempty"`
	TableName      *string  `json:"tablename,omitempty"`
	Postgis        *bool    `json:"postgis,omitempty"`
	MinScale       *string  `json:"minscale,omitempty"`
	MaxScale       *string  `json:"maxscale,omitempty"`
	TifPath        *string  `json:"tif_path,omitempty"`
	Resample       *string  `json:"resample,omitempty"`
	LabelNoClip    bool     `json:"label_no_clip,omitempty"`
}

type WFSLayer struct {
	BaseLayer
}

//nolint:tagliatelle
type GroupLayer struct {
	Name       string `json:"name"`
	Title      string `json:"title"`
	Abstract   string `json:"abstract"`
	StyleName  string `json:"style_name"`
	StyleTitle string `json:"style_title"`
}

//nolint:tagliatelle
type WMSLayer struct {
	BaseLayer
	GroupName                   string  `json:"group_name,omitempty"`
	Styles                      []Style `json:"styles"`
	Offsite                     string  `json:"offsite,omitempty"`
	GetFeatureInfoIncludesClass bool    `json:"get_feature_info_includes_class,omitempty"`
}

type Column struct {
	Name  string  `json:"name"`
	Alias *string `json:"alias,omitempty"`
}

type Style struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}

func SetDataFields[O pdoknlv3.WMSWFS](obj O, wmsLayer *WMSLayer, data pdoknlv3.Data) {
	switch {
	case data.Gpkg != nil:
		gpkg := data.Gpkg

		wmsLayer.GeometryType = &gpkg.GeometryType
		geopackageConstructedPath := "/srv/data/gpkg/" + path.Base(gpkg.BlobKey)
		if !obj.Options().PrefetchData {
			reReplace := regexp.MustCompile(`$[a-zA-Z0-9_]*]/`)
			geopackageConstructedPath = path.Join("/vsiaz/geopackages", reReplace.ReplaceAllString(gpkg.BlobKey, ""))
		}
		wmsLayer.GeopackagePath = &geopackageConstructedPath
	case data.TIF != nil:
		tif := data.TIF
		wmsLayer.GeometryType = smoothoperatorutils.Pointer("Raster")
		wmsLayer.BaseLayer.TifPath = smoothoperatorutils.Pointer(path.Join(tifPath, path.Base(tif.BlobKey)))
		if !obj.Options().PrefetchData {
			reReplace := regexp.MustCompile(`$[a-zA-Z0-9_]*]/`)
			wmsLayer.BaseLayer.TifPath = smoothoperatorutils.Pointer(path.Join("/vsiaz", reReplace.ReplaceAllString(tif.BlobKey, "")))
		}
		wmsLayer.BaseLayer.Resample = &tif.Resample
		wmsLayer.Offsite = smoothoperatorutils.PointerVal(tif.Offsite, "")
	case data.Postgis != nil:
		postgis := data.Postgis
		wmsLayer.Postgis = smoothoperatorutils.Pointer(true)
		wmsLayer.GeometryType = &postgis.GeometryType
	}
}
