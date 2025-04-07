package capabilitiesgenerator

import (
	"fmt"
	"github.com/pdok/ogc-specifications/pkg/wms130"
	"strconv"
	"strings"

	"github.com/cbroglie/mustache"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	capabilitiesgenerator "github.com/pdok/ogc-capabilities-generator/pkg/config"
	"github.com/pdok/ogc-specifications/pkg/wfs200"
	"github.com/pdok/ogc-specifications/pkg/wsc110"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
)

const (
	inspireSchemaLocations  = "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0 http://inspire.ec.europa.eu/schemas/inspire_dls/1.0/inspire_dls.xsd"
	wfsCapabilitiesFilename = "/var/www/config/capabilities_wfs_200.xml"
	wmsCapabilitiesFilename = "/var/www/config/capabilities_wms_130.xml"
	metadataMediaType       = "application/vnd.ogc.csw.GetRecordByIdResponse_xml"
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
			Path:              "/" + pdoknlv3.GetBaseURLPath(wfs),
			Version:           *mapperutils.GetLabelValueByKey(wfs.ObjectMeta.Labels, "service-version"),
		},
		Services: capabilitiesgenerator.Services{
			WFS200Config: &capabilitiesgenerator.WFS200Config{
				Filename: wfsCapabilitiesFilename,
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
		metadataURL, _ := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, wfs.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier)

		config.Services.WFS200Config.Wfs200.Capabilities.OperationsMetadata = &wfs200.OperationsMetadata{
			ExtendedCapabilities: &wfs200.ExtendedCapabilities{
				ExtendedCapabilities: wfs200.NestedExtendedCapabilities{
					MetadataURL: wfs200.MetadataURL{
						URL:       metadataURL,
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

		metadataURL, err := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, fType.DatasetMetadataURL.CSW.MetadataIdentifier)
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
				Href: metadataURL,
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

	return serviceProvider
}

func MapWMSToCapabilitiesGeneratorInput(wms *pdoknlv3.WMS, ownerInfo *smoothoperatorv1.OwnerInfo) (*capabilitiesgenerator.Config, error) {
	abstract := mapperutils.EscapeQuotes(wms.Spec.Service.Abstract)
	var fees *string = nil
	if wms.Spec.Service.Fees != nil {
		feesPtr := mapperutils.EscapeQuotes(*wms.Spec.Service.Fees)
		fees = &feesPtr
	}

	config := capabilitiesgenerator.Config{
		Global: capabilitiesgenerator.Global{
			Namespace:         mapperutils.GetNamespaceURI("prefix", ownerInfo),
			Prefix:            "prefix",
			Onlineresourceurl: pdoknlv3.GetHost(),
			Path:              "/" + pdoknlv3.GetBaseURLPath(wms),
			Version:           *mapperutils.GetLabelValueByKey(wms.ObjectMeta.Labels, "service-version"),
		},
		Services: capabilitiesgenerator.Services{
			WMS130Config: &capabilitiesgenerator.WMS130Config{
				Filename: wmsCapabilitiesFilename,
				Wms130: wms130.GetCapabilitiesResponse{
					WMSService: wms130.WMSService{
						Name:               "WMS",
						Title:              mapperutils.EscapeQuotes(wms.Spec.Service.Title),
						Abstract:           &abstract,
						KeywordList:        &wms130.Keywords{Keyword: wms.Spec.Service.Keywords},
						OnlineResource:     wms130.OnlineResource{Href: &wms.Spec.Service.URL},
						ContactInformation: getContactInformation(ownerInfo),
						Fees:               fees,
						AccessConstraints:  &wms.Spec.Service.AccessConstraints,
						LayerLimit:         nil,
						MaxWidth:           nil,
						MaxHeight:          nil,
					},
					Capabilities: wms130.Capabilities{
						WMSCapabilities: wms130.WMSCapabilities{
							Request: wms130.Request{
								GetCapabilities: wms130.RequestType{},
								GetMap:          wms130.RequestType{},
								GetFeatureInfo:  nil,
							},
							Exception:            wms130.ExceptionType{Format: []string{"XML", "BLANK"}},
							ExtendedCapabilities: nil,
							Layer:                nil,
						},
						OptionalConstraints: wms130.OptionalConstraints{},
					},
				},
			},
		},
	}

	if wms.Spec.Service.Inspire != nil {
		config.Global.AdditionalSchemaLocations = inspireSchemaLocations
		metadataURL, _ := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, wms.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier)

		defaultLanguage := wms130.Language{Language: wms.Spec.Service.Inspire.Language}

		config.Services.WMS130Config.Wms130.Capabilities.ExtendedCapabilities = &wms130.ExtendedCapabilities{
			MetadataURL: wms130.ExtendedMetadataURL{URL: metadataURL, MediaType: metadataMediaType},
			SupportedLanguages: wms130.SupportedLanguages{
				DefaultLanguage:   defaultLanguage,
				SupportedLanguage: &[]wms130.Language{defaultLanguage},
			},
			ResponseLanguage: defaultLanguage,
		}
	}

	return &config, nil
}

func getContactInformation(ownerInfo *smoothoperatorv1.OwnerInfo) *wms130.ContactInformation {
	result := wms130.ContactInformation{
		ContactPersonPrimary:         nil,
		ContactPosition:              nil,
		ContactAddress:               nil,
		ContactVoiceTelephone:        nil,
		ContactFacsimileTelephone:    nil,
		ContactElectronicMailAddress: nil,
	}

	providedContactInformation := ownerInfo.Spec.WMS.ContactInformation

	if providedContactInformation == nil {
		return &result
	}

	if providedContactInformation.ContactPersonPrimary != nil {
		contactPerson := ""
		if providedContactInformation.ContactPersonPrimary.ContactPerson != nil {
			contactPerson = *providedContactInformation.ContactPersonPrimary.ContactPerson
		}
		contactOrganisation := ""
		if providedContactInformation.ContactPersonPrimary.ContactOrganization != nil {
			contactOrganisation = *providedContactInformation.ContactPersonPrimary.ContactOrganization
		}

		contactPersonPrimary := wms130.ContactPersonPrimary{
			ContactPerson:       contactPerson,
			ContactOrganization: contactOrganisation,
		}
		result.ContactPersonPrimary = &contactPersonPrimary
	}

	result.ContactPosition = providedContactInformation.ContactPosition
	if providedContactInformation.ContactAddress != nil {
		contactAddressInput := providedContactInformation.ContactAddress
		contactAddress := wms130.ContactAddress{
			AddressType:     pointerValOrDefault(contactAddressInput.AddressType, ""),
			Address:         pointerValOrDefault(contactAddressInput.Address, ""),
			City:            pointerValOrDefault(contactAddressInput.City, ""),
			StateOrProvince: pointerValOrDefault(contactAddressInput.StateOrProvince, ""),
			PostalCode:      pointerValOrDefault(contactAddressInput.PostCode, ""),
			Country:         pointerValOrDefault(contactAddressInput.Country, ""),
		}
		result.ContactAddress = &contactAddress
	}

	result.ContactVoiceTelephone = providedContactInformation.ContactVoiceTelephone
	result.ContactFacsimileTelephone = providedContactInformation.ContactFacsimileTelephone
	result.ContactElectronicMailAddress = providedContactInformation.ContactElectronicMailAddress

	return &result
}

func pointerValOrDefault[T any](pointer *T, defaultValue T) T {
	if pointer != nil {
		return *pointer
	} else {
		return defaultValue
	}
}

func asPtr[T any](value T) *T {
	return &value
}
