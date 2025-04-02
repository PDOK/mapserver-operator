package capabilitiesgenerator

import (
	"fmt"
	"github.com/cbroglie/mustache"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	capabilitiesgenerator "github.com/pdok/ogc-capabilities-generator/pkg/config"
	"github.com/pdok/ogc-specifications/pkg/wfs200"
	"github.com/pdok/ogc-specifications/pkg/wsc110"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"strconv"
	"strings"
)

const (
	inspireSchemaLocations = "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0 http://inspire.ec.europa.eu/schemas/inspire_dls/1.0/inspire_dls.xsd"
	capabilitiesFilename   = "/var/www/config/capabilities_wfs_200.xml"
	metadataMediaType      = "application/vnd.ogc.csw.GetRecordByIdResponse_xml"
)

func MapWFSToCapabilitiesGeneratorInput(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (*capabilitiesgenerator.Config, error) {
	featureTypeList, err := getFeatureTypeList(wfs, ownerInfo)
	if err != nil {
		return nil, err
	}

	config := capabilitiesgenerator.Config{
		Global: capabilitiesgenerator.Global{
			Namespace:         mapperutils.GetNamespaceURI(wfs.Spec.Service.Prefix, ownerInfo),
			Prefix:            wfs.Spec.Service.Prefix,
			Onlineresourceurl: pdoknlv3.GetHost(),
			Path:              mapperutils.GetPath(wfs),
			Version:           *mapperutils.GetLabelValueByKey(wfs.ObjectMeta.Labels, "service-version"),
		},
		Services: capabilitiesgenerator.Services{
			WFS200Config: &capabilitiesgenerator.WFS200Config{
				Filename: capabilitiesFilename,
				Wfs200: wfs200.GetCapabilitiesResponse{

					ServiceProvider: mapServiceProvider(&ownerInfo.Spec.WFS.ServiceProvider),
					ServiceIdentification: wfs200.ServiceIdentification{
						Title:             mapperutils.EscapeQuotes(wfs.Spec.Service.Title),
						Abstract:          mapperutils.EscapeQuotes(wfs.Spec.Service.Abstract),
						AccessConstraints: wfs.Spec.Service.AccessConstraints,
						Keywords: &wsc110.Keywords{
							Keyword: wfs.Spec.Service.Keywords,
						},
					},
					Capabilities: wfs200.Capabilities{
						FeatureTypeList: *featureTypeList,
					},
				},
			},
		},
	}

	if wfs.Spec.Service.Inspire != nil {
		config.Global.AdditionalSchemaLocations = inspireSchemaLocations
		metadataUrl, _ := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, wfs.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier)

		config.Services.WFS200Config.Wfs200.Capabilities.OperationsMetadata = &wfs200.OperationsMetadata{
			ExtendedCapabilities: &wfs200.ExtendedCapabilities{
				ExtendedCapabilities: wfs200.NestedExtendedCapabilities{
					MetadataURL: wfs200.MetadataURL{
						URL:       metadataUrl,
						MediaType: metadataMediaType,
					},
					SupportedLanguages: wfs200.SupportedLanguages{
						DefaultLanguage: wfs200.Language{
							Language: wfs.Spec.Service.Inspire.Language,
						},
					},
					ResponseLanguage: wfs200.Language{Language: wfs.Spec.Service.Inspire.Language},
					SpatialDataSetIdentifier: wfs200.SpatialDataSetIdentifier{
						Code: wfs.Spec.Service.Inspire.SpatialDatasetIdentifier,
					},
				},
			},
		}
	}

	return &config, nil
}

func getFeatureTypeList(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (*wfs200.FeatureTypeList, error) {
	typeList := wfs200.FeatureTypeList{}

	for _, fType := range wfs.Spec.Service.FeatureTypes {
		defaultCRS, err := createCRSFromEpsgString(wfs.Spec.Service.DefaultCrs)
		if err != nil {
			return nil, err
		}

		var otherCRS []*wfs200.CRS
		for _, epsgString := range wfs.Spec.Service.OtherCrs {
			CRS, err := createCRSFromEpsgString(epsgString)
			if err != nil {
				return nil, err
			}
			otherCRS = append(otherCRS, CRS)
		}

		metadataUrl, err := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, fType.DatasetMetadataURL.CSW.MetadataIdentifier)
		if err != nil {
			return nil, err
		}

		featureType := wfs200.FeatureType{
			Name:     wfs.Spec.Service.Prefix + fType.Name,
			Title:    mapperutils.EscapeQuotes(fType.Title),
			Abstract: mapperutils.EscapeQuotes(fType.Abstract),
			Keywords: &[]wsc110.Keywords{
				{
					Keyword: fType.Keywords,
				},
			},
			MetadataURL: wfs200.MetadataHref{
				Href: metadataUrl,
			},
			DefaultCRS: defaultCRS,
			OtherCRS:   otherCRS,
		}

		typeList.FeatureType = append(typeList.FeatureType, featureType)
	}
	return &typeList, nil
}

func createCRSFromEpsgString(epsgString string) (*wfs200.CRS, error) {
	index := strings.LastIndex(epsgString, ":")
	if index == -1 {
		return nil, fmt.Errorf("could not determine EPSG code from EPSG string %s", epsgString)
	}
	epsgCodeString := epsgString[index+1:]
	epsgCode, err := strconv.Atoi(epsgCodeString)
	if err != nil {
		return nil, fmt.Errorf("could not determine EPSG code from EPSG string %s", epsgCodeString)
	}

	return &wfs200.CRS{
		Code: epsgCode,
	}, nil
}

func replaceMustachTemplate(hrefTemplate string, identifier string) (string, error) {
	templateVariable := map[string]string{"identifier": identifier}
	return mustache.Render(hrefTemplate, templateVariable)
}

func mapServiceProvider(provider *smoothoperatorv1.ServiceProvider) (serviceProvider wfs200.ServiceProvider) {
	if provider.ProviderName != nil {
		serviceProvider.ProviderName = provider.ProviderName
	}

	if provider.ProviderSite != nil {
		serviceProvider.ProviderSite = &wfs200.ProviderSite{
			Type: provider.ProviderSite.Type,
			Href: provider.ProviderSite.Href,
		}
	}

	if provider.ServiceContact != nil {
		serviceProvider.ServiceContact = &wfs200.ServiceContact{
			IndividualName: provider.ServiceContact.IndividualName,
			PositionName:   provider.ServiceContact.PositionName,
			Role:           provider.ServiceContact.Role,
		}
		if provider.ServiceContact.ContactInfo != nil {
			serviceProvider.ServiceContact.ContactInfo = &wfs200.ContactInfo{
				Text:                provider.ServiceContact.ContactInfo.Text,
				HoursOfService:      provider.ServiceContact.ContactInfo.HoursOfService,
				ContactInstructions: provider.ServiceContact.ContactInfo.ContactInstructions,
			}
			if provider.ServiceContact.ContactInfo.Phone != nil {
				serviceProvider.ServiceContact.ContactInfo.Phone = &wfs200.Phone{
					Voice:     provider.ServiceContact.ContactInfo.Phone.Voice,
					Facsimile: provider.ServiceContact.ContactInfo.Phone.Facsimile,
				}
			}
			if provider.ServiceContact.ContactInfo.Address != nil {
				serviceProvider.ServiceContact.ContactInfo.Address = &wfs200.Address{
					DeliveryPoint:         provider.ServiceContact.ContactInfo.Address.DeliveryPoint,
					City:                  provider.ServiceContact.ContactInfo.Address.City,
					AdministrativeArea:    provider.ServiceContact.ContactInfo.Address.AdministrativeArea,
					PostalCode:            provider.ServiceContact.ContactInfo.Address.PostalCode,
					Country:               provider.ServiceContact.ContactInfo.Address.Country,
					ElectronicMailAddress: provider.ServiceContact.ContactInfo.Address.ElectronicMailAddress,
				}
			}
			if provider.ServiceContact.ContactInfo.OnlineResource != nil {
				serviceProvider.ServiceContact.ContactInfo.OnlineResource = &wfs200.OnlineResource{
					Type: provider.ServiceContact.ContactInfo.OnlineResource.Type,
					Href: provider.ServiceContact.ContactInfo.OnlineResource.Href,
				}
			}
		}
	}
	return
}
