package capabilitiesgenerator

import (
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	"github.com/pdok/ogc-specifications/pkg/wfs200"
	"github.com/pdok/ogc-specifications/pkg/wsc110"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
)

const (
	inspireSchemaLocations = "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0 http://inspire.ec.europa.eu/schemas/inspire_dls/1.0/inspire_dls.xsd"
	capabilitiesFilename   = "/var/www/config/capabilities_wfs_200.xml"
)

func MapWFSToCapabilitiesGeneratorInput(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (Config, error) {
	config := Config{
		Global: Global{
			Namespace:         mapperutils.GetNamespaceURI(wfs.Spec.Service.Prefix, ownerInfo),
			Prefix:            wfs.Spec.Service.Prefix,
			OnlineResourceurl: pdoknlv3.GetHost(),
			Path:              mapperutils.GetPath(wfs),
			Version:           *mapperutils.GetLabelValueByKey(wfs.ObjectMeta.Labels, "service-version"),
		},
		Services: Services{
			WFS200Config: &WFS200Config{
				Filename: capabilitiesFilename,
				Wfs200: wfs200.GetCapabilitiesResponse{

					ServiceProvider: wfs200.ServiceProvider{
						ProviderSite: struct {
							Type string `xml:"xlink:type,attr" yaml:"type"`
							Href string `xml:"xlink:href,attr" yaml:"href"`
						}(struct {
							Type string
							Href string
						}{
							Type: "simple",
							Href: pdoknlv3.GetHost(),
						}),
					},
					ServiceIdentification: wfs200.ServiceIdentification{
						Title:             mapperutils.EscapeQuotes(wfs.Spec.Service.Title),
						Abstract:          mapperutils.EscapeQuotes(wfs.Spec.Service.Abstract),
						AccessConstraints: wfs.Spec.Service.AccessConstraints,
						Keywords: &wsc110.Keywords{
							Keyword: wfs.Spec.Service.Keywords,
						},
					},
					Capabilities: wfs200.Capabilities{
						FeatureTypeList: getFeatureTypeList(wfs),
					},
				},
			},
		},
	}

	if wfs.Spec.Service.Inspire != nil {
		config.Global.AdditionalSchemaLocations = inspireSchemaLocations

		// Todo set extended capabilities
		//config.Services.WFS200Config.Wfs200.Capabilities.OperationsMetadata = wfs200.OperationsMetadata{}
	}

	return config, nil
}

func getFeatureTypeList(wfs *pdoknlv3.WFS) (typeList wfs200.FeatureTypeList) {
	typeList.FeatureType = []wfs200.FeatureType{}

	for _, fType := range wfs.Spec.Service.FeatureTypes {
		featureType := wfs200.FeatureType{
			Name:       wfs.Spec.Service.Prefix + fType.Name,
			Title:      mapperutils.EscapeQuotes(fType.Title),
			Abstract:   mapperutils.EscapeQuotes(fType.Abstract),
			DefaultCRS: &wfs200.CRS{Namespace: "", Code: 1}, // Todo
			OtherCRS: &[]wfs200.CRS{ // Todo
				{Namespace: "", Code: 2},
			},
		}
		typeList.FeatureType = append(typeList.FeatureType, featureType)
	}
	return
}
