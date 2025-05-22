package constants

const (
	MapserverName             = "mapserver"
	OgcWebserviceProxyName    = "ogc-webservice-proxy"
	MapfileGeneratorName      = "mapfile-generator"
	CapabilitiesGeneratorName = "capabilities-generator"
	BlobDownloadName          = "blob-download"
	InitScriptsName           = "init-scripts"
	LegendGeneratorName       = "legend-generator"
	LegendFixerName           = "legend-fixer"
	FeatureinfoGeneratorName  = "featureinfo-generator"

	BaseVolumeName = "base"
	DataVolumeName = "data"

	configSuffix                             = "-config"
	ConfigMapMapfileGeneratorVolumeName      = MapfileGeneratorName + configSuffix
	ConfigMapStylingFilesVolumeName          = "styling-files"
	ConfigMapCapabilitiesGeneratorVolumeName = CapabilitiesGeneratorName + configSuffix
	ConfigMapOgcWebserviceProxyVolumeName    = OgcWebserviceProxyName + configSuffix
	ConfigMapLegendGeneratorVolumeName       = LegendGeneratorName + configSuffix
	ConfigMapFeatureinfoGeneratorVolumeName  = FeatureinfoGeneratorName + configSuffix
	ConfigMapCustomMapfileVolumeName         = "mapfile"

	HTMLTemplatesPath       = "/srv/data/config/templates"
	ApachePortNr      int32 = 9117
)
