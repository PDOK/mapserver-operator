package capabilitiesgenerator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pdok/ogc-specifications/pkg/wms130"

	"github.com/cbroglie/mustache"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	capabilitiesgenerator "github.com/pdok/ogc-capabilities-generator/pkg/config"
	"github.com/pdok/ogc-specifications/pkg/wfs200"
	"github.com/pdok/ogc-specifications/pkg/wsc110"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

const (
	inspireSchemaLocationsWFS = "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0 http://inspire.ec.europa.eu/schemas/inspire_dls/1.0/inspire_dls.xsd"
	inspireSchemaLocationsWMS = "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0 http://inspire.ec.europa.eu/schemas/inspire_dls/1.0/inspire_dls.xsd http://inspire.ec.europa.eu/schemas/common/1.0 http://inspire.ec.europa.eu/schemas/common/1.0/common.xsd"
	wfsCapabilitiesFilename   = "/var/www/config/capabilities_wfs_200.xml"
	wmsCapabilitiesFilename   = "/var/www/config/capabilities_wms_130.xml"
	metadataMediaType         = "application/vnd.ogc.csw.GetRecordByIdResponse_xml"
	XLinkURL                  = "http://www.w3.org/1999/xlink"
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
			Onlineresourceurl: pdoknlv3.GetHost(true),
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
						AccessConstraints: *wfs.Spec.Service.AccessConstraints,
						Keywords: &wsc110.Keywords{
							Keyword: wfs.Spec.Service.KeywordsIncludingInspireKeyword(),
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
		config.Global.AdditionalSchemaLocations = inspireSchemaLocationsWFS
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
			Name:     wfs.Spec.Service.Prefix + ":" + fType.Name,
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

	// TODO fix linting (nestif)
	//nolint:nestif
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
	canonicalServiceURL := pdoknlv3.GetHost(true) + "/" + pdoknlv3.GetBaseURLPath(wms)

	abstract := mapperutils.EscapeQuotes(wms.Spec.Service.Abstract)

	maxWidth := 4000
	maxHeight := 4000

	config := capabilitiesgenerator.Config{
		Global: capabilitiesgenerator.Global{
			// Prefix is unused for the WMS
			Namespace:         mapperutils.GetNamespaceURI("", ownerInfo),
			Prefix:            "",
			Onlineresourceurl: pdoknlv3.GetHost(true),
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
						KeywordList:        &wms130.Keywords{Keyword: wms.Spec.Service.KeywordsIncludingInspireKeyword()},
						OnlineResource:     wms130.OnlineResource{Href: smoothoperatorutils.Pointer(pdoknlv3.GetHost(true))},
						ContactInformation: getContactInformation(ownerInfo),
						Fees:               smoothoperatorutils.Pointer("NONE"),
						AccessConstraints:  &wms.Spec.Service.AccessConstraints,
						OptionalConstraints: &wms130.OptionalConstraints{
							MaxWidth:  maxWidth,
							MaxHeight: maxHeight,
						},
					},
					Capabilities: wms130.Capabilities{
						WMSCapabilities: wms130.WMSCapabilities{
							Request: wms130.Request{
								GetCapabilities: wms130.RequestType{
									Format:  []string{"text/xml"},
									DCPType: getDcpType(canonicalServiceURL, false),
								},
								GetMap: wms130.RequestType{
									Format:  []string{"image/png", "image/jpeg", "image/png; mode=8bit", "image/vnd.jpeg-png", "image/vnd.jpeg-png8"},
									DCPType: getDcpType(canonicalServiceURL, true),
								},
								GetFeatureInfo: &wms130.RequestType{
									Format:  []string{"application/json", "application/json; subtype=geojson", "application/vnd.ogc.gml", "text/html", "text/plain", "text/xml", "text/xml; subtype=gml/3.1.1"},
									DCPType: getDcpType(canonicalServiceURL, true),
								},
							},
							Exception:            wms130.ExceptionType{Format: []string{"XML", "BLANK"}},
							ExtendedCapabilities: nil,
							Layer:                getLayers(wms, canonicalServiceURL),
						},
						OptionalConstraints: nil,
					},
				},
			},
		},
	}

	if wms.Spec.Service.Inspire != nil {
		config.Global.AdditionalSchemaLocations = inspireSchemaLocationsWMS
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
			AddressType:     smoothoperatorutils.PointerVal(contactAddressInput.AddressType, ""),
			Address:         smoothoperatorutils.PointerVal(contactAddressInput.Address, ""),
			City:            smoothoperatorutils.PointerVal(contactAddressInput.City, ""),
			StateOrProvince: smoothoperatorutils.PointerVal(contactAddressInput.StateOrProvince, ""),
			PostalCode:      smoothoperatorutils.PointerVal(contactAddressInput.PostCode, ""),
			Country:         smoothoperatorutils.PointerVal(contactAddressInput.Country, ""),
		}
		result.ContactAddress = &contactAddress
	}

	result.ContactVoiceTelephone = providedContactInformation.ContactVoiceTelephone
	result.ContactFacsimileTelephone = providedContactInformation.ContactFacsimileTelephone
	result.ContactElectronicMailAddress = providedContactInformation.ContactElectronicMailAddress

	return &result
}

func getDcpType(url string, fillPost bool) *wms130.DCPType {
	get := wms130.Method{
		OnlineResource: wms130.OnlineResource{
			Xlink: smoothoperatorutils.Pointer(XLinkURL),
			Type:  nil,
			Href:  smoothoperatorutils.Pointer(url),
		},
	}

	var post *wms130.Method
	if fillPost {
		post = &get
	}

	result := wms130.DCPType{
		HTTP: struct {
			Get  wms130.Method  `xml:"Get" yaml:"get"`
			Post *wms130.Method `xml:"Post" yaml:"post"`
		}{
			Get:  get,
			Post: post,
		},
	}
	return &result
}

// TODO fix linting (funlen)
//
//nolint:funlen
func getLayers(wms *pdoknlv3.WMS, canonicalURL string) []wms130.Layer {
	result := make([]wms130.Layer, 0)
	referenceLayer := wms.Spec.Service.Layer
	title := referenceLayer.Title
	if title != nil {
		title = smoothoperatorutils.Pointer(mapperutils.EscapeQuotes(*referenceLayer.Title))
	} else {
		title = smoothoperatorutils.Pointer("")
	}

	var authorityURL *wms130.AuthorityURL
	var identifier *wms130.Identifier

	if referenceLayer.Authority != nil {
		authorityURL = &wms130.AuthorityURL{
			Name: referenceLayer.Authority.Name,
			OnlineResource: wms130.OnlineResource{
				Xlink: smoothoperatorutils.Pointer(XLinkURL),
				Type:  nil,
				Href:  &referenceLayer.Authority.URL,
			},
		}
		identifier = &wms130.Identifier{
			Authority: referenceLayer.Authority.Name,
			Value:     referenceLayer.Authority.SpatialDatasetIdentifier,
		}
	}

	topLayer := wms130.Layer{
		Queryable:   smoothoperatorutils.Pointer(1),
		Opaque:      nil,
		Name:        nil,
		Title:       *title,
		Abstract:    smoothoperatorutils.Pointer(mapperutils.EscapeQuotes(wms.Spec.Service.Abstract)),
		KeywordList: &wms130.Keywords{Keyword: referenceLayer.Keywords},
		//CRS:                     getDefaultWMSCRSes(),
		//EXGeographicBoundingBox: &defaultWMSBoundingBox,
		//BoundingBox:             getDefaultWMSLayerBoundingBoxes(),
		Dimension:           nil,
		Attribution:         nil,
		AuthorityURL:        authorityURL,
		Identifier:          identifier,
		MetadataURL:         nil,
		DataURL:             nil,
		FeatureListURL:      nil,
		Style:               nil,
		MinScaleDenominator: nil,
		MaxScaleDenominator: nil,
		Layer:               []*wms130.Layer{},
	}

	for _, layer := range referenceLayer.Layers {
		var minScaleDenom *float64
		var maxScaleDenom *float64
		var innerIdentifier *wms130.Identifier
		var innerAuthorityURL *wms130.AuthorityURL
		metadataUrls := make([]*wms130.MetadataURL, 0)

		if layer.MinScaleDenominator != nil {
			float, err := strconv.ParseFloat(*layer.MinScaleDenominator, 64)
			if err == nil {
				minScaleDenom = &float
			}
		}

		if layer.MaxScaleDenominator != nil {
			float, err := strconv.ParseFloat(*layer.MaxScaleDenominator, 64)
			if err == nil {
				maxScaleDenom = &float
			}
		}

		if layer.DatasetMetadataURL != nil {
			metadataUrls = append(metadataUrls, &wms130.MetadataURL{
				Type:   smoothoperatorutils.Pointer("TC211"),
				Format: smoothoperatorutils.Pointer("text/plain"),
				OnlineResource: wms130.OnlineResource{
					Xlink: smoothoperatorutils.Pointer(XLinkURL),
					Type:  smoothoperatorutils.Pointer("simple"),
					Href:  smoothoperatorutils.Pointer("https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=" + layer.DatasetMetadataURL.CSW.MetadataIdentifier),
				},
			})
		}

		if layer.Authority != nil {
			innerAuthorityURL = &wms130.AuthorityURL{
				Name: layer.Authority.Name,
				OnlineResource: wms130.OnlineResource{
					Xlink: smoothoperatorutils.Pointer(XLinkURL),
					Type:  nil,
					Href:  &layer.Authority.URL,
				},
			}
			innerIdentifier = &wms130.Identifier{
				Authority: layer.Authority.Name,
				Value:     layer.Authority.SpatialDatasetIdentifier,
			}
		}

		nestedLayer := wms130.Layer{
			Queryable: smoothoperatorutils.Pointer(1),
			Opaque:    nil,
			Name:      layer.Name,
			Title:     smoothoperatorutils.PointerVal(layer.Title, ""),
			Abstract:  layer.Abstract,
			KeywordList: &wms130.Keywords{
				Keyword: layer.Keywords,
			},
			//CRS:                     getDefaultWMSCRSes(),
			//EXGeographicBoundingBox: &defaultWMSBoundingBox,
			//BoundingBox:             getDefaultWMSLayerBoundingBoxes(),
			Dimension:           nil,
			Attribution:         nil,
			AuthorityURL:        innerAuthorityURL,
			Identifier:          innerIdentifier,
			MetadataURL:         metadataUrls,
			DataURL:             nil,
			FeatureListURL:      nil,
			Style:               []*wms130.Style{},
			MinScaleDenominator: minScaleDenom,
			MaxScaleDenominator: maxScaleDenom,
			Layer:               nil,
		}
		for _, style := range layer.Styles {
			newStyle := wms130.Style{
				Name:     style.Name,
				Title:    smoothoperatorutils.PointerVal(style.Title, ""),
				Abstract: style.Abstract,
				LegendURL: &wms130.LegendURL{
					Width:  78,
					Height: 20,
					Format: "image/png",
					OnlineResource: wms130.OnlineResource{
						Xlink: smoothoperatorutils.Pointer(XLinkURL),
						Type:  smoothoperatorutils.Pointer("simple"),
						Href:  smoothoperatorutils.Pointer(canonicalURL + "/legend/" + *layer.Name + "/" + style.Name + ".png"),
					},
				},
				StyleSheetURL: nil,
			}
			nestedLayer.Style = append(nestedLayer.Style, &newStyle)
		}

		topLayer.Layer = append(topLayer.Layer, &nestedLayer)
	}

	result = append(result, topLayer)
	return result
}
