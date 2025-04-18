package mapfilegenerator

//nolint:tagliatelle
type BaseServiceInput struct {
	Title           string   `json:"service_title"`
	Abstract        string   `json:"service_abstract"`
	Keywords        string   `json:"service_keywords"`
	Extent          string   `json:"service_extent"`
	NamespacePrefix string   `json:"service_namespace_prefix"`
	NamespaceURI    string   `json:"service_namespace_uri"`
	OnlineResource  string   `json:"service_onlineresource"`
	Path            string   `json:"service_path"`
	MetadataId      string   `json:"service_metadata_id"`
	DatasetOwner    *string  `json:"dataset_owner,omitempty"`
	AuthorityURL    *string  `json:"authority_url,omitempty"`
	AutomaticCasing bool     `json:"automatic_casing"`
	DataEPSG        string   `json:"data_epsg"`
	EPSGList        []string `json:"epsg_list"`
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
	AccessConstraints string     `json:"service_accessconstraints"`
	Layers            []WMSLayer `json:"layers"`
	Templates         string     `json:"templates,omitempty"`
}

//nolint:tagliatelle
type BaseLayer struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Abstract       string   `json:"abstract"`
	Keywords       string   `json:"keywords"`
	Extent         string   `json:"layer_extent"`
	MetadataId     string   `json:"dataset_metadata_id"`
	Columns        []Column `json:"columns"`
	GeometryType   *string  `json:"geometry_type,omitempty"`
	GeopackagePath *string  `json:"gpkg_path,omitempty"`
	TableName      *string  `json:"tablename,omitempty"`
	Postgis        *bool    `json:"postgis,omitempty"`
}

type WFSLayer struct {
	BaseLayer
}

type WMSLayer struct {
	BaseLayer
	GroupName                   string  `json:"group_name,omitempty"`
	Styles                      []Style `json:"styles"`
	Offsite                     string  `json:"offsite,omitempty"`
	GetFeatureInfoIncludesClass bool    `json:"get_feature_info_includes_class"`
}

type Column struct {
	Name  string  `json:"name"`
	Alias *string `json:"alias,omitempty"`
}

type Style struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}
