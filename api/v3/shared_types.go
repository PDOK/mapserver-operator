package v3

type Options struct {
	AutomaticCasing bool `json:"automaticCasing"`
	PrefetchData    bool `json:"prefetchData"`
	IncludeIngress  bool `json:"includeIngress"`
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

type Columns struct {
	Name  string  `json:"name"`
	Alias *string `json:"alias,omitempty"`
}
