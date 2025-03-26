package mapfilegenerator

type Input struct {
	Title             string   `json:"service_title"`
	Abstract          string   `json:"service_abstract"`
	Keywords          string   `json:"service_keywords"`
	AccessConstraints string   `json:"service_accessconstraints"`
	Extent            string   `json:"service_extent"`
	WFSMaxFeatures    string   `json:"service_wfs_maxfeatures"`
	NamespacePrefix   string   `json:"service_namespace_prefix"`
	NamespaceURI      string   `json:"service_namespace_uri"`
	OnlineResource    string   `json:"service_onlineresource"`
	Path              string   `json:"service_path"`
	MetadataId        string   `json:"service_metadata_id"`
	DatasetOwner      *string  `json:"dataset_owner,omitempty"`
	AuthorityURL      *string  `json:"authority_url,omitempty"`
	AutomaticCasing   bool     `json:"automatic_casing"`
	DataEPSG          string   `json:"data_epsg"`
	EPSGList          []string `json:"epsg_list"`
	Layers            []Layer  `json:"layers"`
}

type Layer struct {
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

type Column struct {
	Name  string  `json:"name"`
	Alias *string `json:"alias,omitempty"`
}
