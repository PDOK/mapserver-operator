package types

type HashedConfigMapNames struct {
	Mapserver             string
	InitScripts           string
	MapfileGenerator      string
	CapabilitiesGenerator string
	OgcWebserviceProxy    string
	LegendGenerator       string
	FeatureInfoGenerator  string
}

type Images struct {
	MapserverImage             string
	MultitoolImage             string
	MapfileGeneratorImage      string
	CapabilitiesGeneratorImage string
	FeatureinfoGeneratorImage  string
	OgcWebserviceProxyImage    string
	ApacheExporterImage        string
}
