package capabilitiesgenerator

import (
	"fmt"
	"slices"
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

	serviceVersion := mapperutils.GetLabelValueByKey(wfs.ObjectMeta.Labels, "service-version")
	if serviceVersion == nil {
		serviceVersion = mapperutils.GetLabelValueByKey(wfs.ObjectMeta.Labels, "pdok.nl/service-version")
	}
	config := capabilitiesgenerator.Config{
		Global: capabilitiesgenerator.Global{
			Namespace:         mapperutils.GetNamespaceURI(wfs.Spec.Service.Prefix, ownerInfo),
			Prefix:            wfs.Spec.Service.Prefix,
			Onlineresourceurl: wfs.URL().Scheme + "://" + wfs.URL().Host,
			Path:              wfs.URL().Path,
			Version:           *serviceVersion,
		},
		Services: capabilitiesgenerator.Services{
			WFS200Config: &capabilitiesgenerator.WFS200Config{
				Filename: wfsCapabilitiesFilename,
				Wfs200: wfs200.GetCapabilitiesResponse{

					ServiceProvider: mapServiceProvider(&ownerInfo.Spec.WFS.ServiceProvider),
					ServiceIdentification: wfs200.ServiceIdentification{
						Title:             mapperutils.EscapeQuotes(wfs.Spec.Service.Title),
						Abstract:          mapperutils.EscapeQuotes(wfs.Spec.Service.Abstract),
						AccessConstraints: wfs.Spec.Service.AccessConstraints.String(),
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
		var metadataURL wfs200.MetadataURL
		if wfs.Spec.Service.Inspire.ServiceMetadataURL.CSW != nil {
			metadataURL.URL, err = replaceMustacheTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, wfs.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier)
			if err != nil {
				return nil, err
			}
			metadataURL.MediaType = metadataMediaType
		}

		if wfs.Spec.Service.Inspire.ServiceMetadataURL.Custom != nil {
			metadataURL.URL = wfs.Spec.Service.Inspire.ServiceMetadataURL.Custom.Href.String()
			metadataURL.MediaType = wfs.Spec.Service.Inspire.ServiceMetadataURL.Custom.Type
		}

		config.Services.WFS200Config.Wfs200.Capabilities.OperationsMetadata = &wfs200.OperationsMetadata{
			ExtendedCapabilities: &wfs200.ExtendedCapabilities{
				ExtendedCapabilities: wfs200.NestedExtendedCapabilities{
					MetadataURL: metadataURL,
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
	if wfs.Spec.Service.CountDefault != nil {
		operationsMetadata := config.Services.WFS200Config.Wfs200.Capabilities.OperationsMetadata
		if operationsMetadata == nil {
			operationsMetadata = &wfs200.OperationsMetadata{}
		}
		operationsMetadata.Constraint = append(operationsMetadata.Constraint, wfs200.Constraint{
			Name:         "CountDefault",
			DefaultValue: smoothoperatorutils.Pointer(strconv.Itoa(*wfs.Spec.Service.CountDefault)),
		})
		config.Services.WFS200Config.Wfs200.Capabilities.OperationsMetadata = operationsMetadata
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

		var wgs84BoundingBox *wsc110.WGS84BoundingBox
		if fType.Bbox != nil && fType.Bbox.WGS84 != nil {
			minX, err := strconv.ParseFloat(fType.Bbox.WGS84.MinX, 64)
			if err != nil {
				return nil, err
			}
			maxX, err := strconv.ParseFloat(fType.Bbox.WGS84.MaxX, 64)
			if err != nil {
				return nil, err
			}
			minY, err := strconv.ParseFloat(fType.Bbox.WGS84.MinY, 64)
			if err != nil {
				return nil, err
			}
			maxY, err := strconv.ParseFloat(fType.Bbox.WGS84.MaxY, 64)
			if err != nil {
				return nil, err
			}

			wgs84BoundingBox = &wsc110.WGS84BoundingBox{
				LowerCorner: wsc110.Position{minX, minY},
				UpperCorner: wsc110.Position{maxX, maxY},
			}
		}

		metadataURL, err := replaceMustacheTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, fType.DatasetMetadataURL.CSW.MetadataIdentifier)
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
			DefaultCRS:       defaultCRS,
			OtherCRS:         otherCRS,
			WGS84BoundingBox: wgs84BoundingBox,
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

func replaceMustacheTemplate(hrefTemplate string, identifier string) (string, error) {
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
			serviceProvider.ServiceContact.ContactInfo = mapContactInfo(*provider.ServiceContact.ContactInfo)
		}
	}

	return serviceProvider
}

func mapContactInfo(contactInfo smoothoperatorv1.ContactInfo) (serviceContactInfo *wfs200.ContactInfo) {
	serviceContactInfo = &wfs200.ContactInfo{
		Text:                contactInfo.Text,
		HoursOfService:      contactInfo.HoursOfService,
		ContactInstructions: contactInfo.ContactInstructions,
	}
	if contactInfo.Phone != nil {
		serviceContactInfo.Phone = &wfs200.Phone{
			Voice:     contactInfo.Phone.Voice,
			Facsimile: contactInfo.Phone.Facsimile,
		}
	}
	if contactInfo.Address != nil {
		serviceContactInfo.Address = &wfs200.Address{
			DeliveryPoint:         contactInfo.Address.DeliveryPoint,
			City:                  contactInfo.Address.City,
			AdministrativeArea:    contactInfo.Address.AdministrativeArea,
			PostalCode:            contactInfo.Address.PostalCode,
			Country:               contactInfo.Address.Country,
			ElectronicMailAddress: contactInfo.Address.ElectronicMailAddress,
		}
	}
	if contactInfo.OnlineResource != nil {
		serviceContactInfo.OnlineResource = &wfs200.OnlineResource{
			Type: contactInfo.OnlineResource.Type,
			Href: contactInfo.OnlineResource.Href,
		}
	}
	return
}

func MapWMSToCapabilitiesGeneratorInput(wms *pdoknlv3.WMS, ownerInfo *smoothoperatorv1.OwnerInfo) (*capabilitiesgenerator.Config, error) {
	canonicalServiceURL := wms.URL()

	abstract := mapperutils.EscapeQuotes(wms.Spec.Service.Abstract)

	serviceVersion := mapperutils.GetLabelValueByKey(wms.ObjectMeta.Labels, "service-version")
	if serviceVersion == nil {
		serviceVersion = mapperutils.GetLabelValueByKey(wms.ObjectMeta.Labels, "pdok.nl/service-version")
	}

	config := capabilitiesgenerator.Config{
		Global: capabilitiesgenerator.Global{
			// Prefix is unused for the WMS, but doesn't hurt to pass it
			Namespace:         mapperutils.GetNamespaceURI(wms.Spec.Service.Prefix, ownerInfo),
			Prefix:            wms.Spec.Service.Prefix,
			Onlineresourceurl: wms.URL().Scheme + "://" + wms.URL().Host,
			Path:              wms.URL().Path,
			Version:           *serviceVersion,
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
						OnlineResource:     wms130.OnlineResource{Href: smoothoperatorutils.Pointer(wms.URL().Scheme + "://" + wms.URL().Host)},
						ContactInformation: getContactInformation(ownerInfo),
						Fees:               smoothoperatorutils.Pointer("NONE"),
						AccessConstraints:  &wms.Spec.Service.AccessConstraints,
						OptionalConstraints: &wms130.OptionalConstraints{
							MaxWidth:  int(smoothoperatorutils.PointerVal(wms.Spec.Service.MaxSize, 4000)),
							MaxHeight: int(smoothoperatorutils.PointerVal(wms.Spec.Service.MaxSize, 4000)),
						},
					},
					Capabilities: wms130.Capabilities{
						WMSCapabilities: wms130.WMSCapabilities{
							Request: wms130.Request{
								GetCapabilities: wms130.RequestType{
									Format:  []string{"text/xml"},
									DCPType: getDcpType(canonicalServiceURL.String(), false),
								},
								GetMap: wms130.RequestType{
									Format:  []string{"image/png", "image/jpeg", "image/png; mode=8bit", "image/vnd.jpeg-png", "image/vnd.jpeg-png8"},
									DCPType: getDcpType(canonicalServiceURL.String(), true),
								},
								GetFeatureInfo: &wms130.RequestType{
									Format:  []string{"application/json", "application/json; subtype=geojson", "application/vnd.ogc.gml", "text/html", "text/plain", "text/xml", "text/xml; subtype=gml/3.1.1"},
									DCPType: getDcpType(canonicalServiceURL.String(), true),
								},
							},
							Exception:            wms130.ExceptionType{Format: []string{"XML", "BLANK"}},
							ExtendedCapabilities: nil,
							Layer:                getLayers(wms, canonicalServiceURL.String()),
						},
						OptionalConstraints: nil,
					},
				},
			},
		},
	}

	if wms.Spec.Service.Inspire != nil {
		config.Global.AdditionalSchemaLocations = inspireSchemaLocationsWMS
		metadataURL, _ := replaceMustacheTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, wms.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier)

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

func getLayers(wms *pdoknlv3.WMS, canonicalURL string) []wms130.Layer {
	return []wms130.Layer{
		mapLayer(wms.Spec.Service.Layer, canonicalURL, nil, nil, nil),
	}
}

func mapLayer(layer pdoknlv3.Layer, canonicalURL string, authorityURL *wms130.AuthorityURL, identifier *wms130.Identifier, parentStyleNames []string) wms130.Layer {
	if layer.Authority != nil {
		authorityURL = &wms130.AuthorityURL{
			Name: layer.Authority.Name,
			OnlineResource: wms130.OnlineResource{
				Xlink: smoothoperatorutils.Pointer(XLinkURL),
				Type:  nil,
				Href:  &layer.Authority.URL,
			},
		}
		identifier = &wms130.Identifier{
			Authority: layer.Authority.Name,
			Value:     layer.Authority.SpatialDatasetIdentifier,
		}
	}

	l := wms130.Layer{
		Queryable:   smoothoperatorutils.Pointer(1),
		Opaque:      nil,
		Name:        layer.Name,
		Title:       mapperutils.EscapeQuotes(smoothoperatorutils.PointerVal(layer.Title, "")),
		Abstract:    smoothoperatorutils.Pointer(mapperutils.EscapeQuotes(smoothoperatorutils.PointerVal(layer.Abstract, ""))),
		KeywordList: &wms130.Keywords{Keyword: layer.Keywords},
		// TODO
		//CRS:                     defaultCrs,
		//EXGeographicBoundingBox: &defaultBoundingBox,
		//BoundingBox:             allDefaultBoundingBoxes,
		Dimension:      nil,
		Attribution:    nil,
		AuthorityURL:   authorityURL,
		Identifier:     identifier,
		DataURL:        nil,
		FeatureListURL: nil,
		Style:          getLayerStyles(layer, canonicalURL, parentStyleNames),
		Layer:          []*wms130.Layer{},
	}

	if layer.MinScaleDenominator != nil {
		float, err := strconv.ParseFloat(*layer.MinScaleDenominator, 64)
		if err == nil {
			l.MinScaleDenominator = &float
		}
	}

	if layer.MaxScaleDenominator != nil {
		float, err := strconv.ParseFloat(*layer.MaxScaleDenominator, 64)
		if err == nil {
			l.MaxScaleDenominator = &float
		}
	}

	if layer.DatasetMetadataURL != nil {
		l.MetadataURL = append(l.MetadataURL, &wms130.MetadataURL{
			Type:   smoothoperatorutils.Pointer("TC211"),
			Format: smoothoperatorutils.Pointer("text/plain"),
			OnlineResource: wms130.OnlineResource{
				Xlink: smoothoperatorutils.Pointer(XLinkURL),
				Type:  smoothoperatorutils.Pointer("simple"),
				Href:  smoothoperatorutils.Pointer("https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=" + layer.DatasetMetadataURL.CSW.MetadataIdentifier),
			},
		})
	}

	layerStyleNames := []string{}
	for _, s := range l.Style {
		layerStyleNames = append(layerStyleNames, s.Name)
	}

	// Map sublayers
	for _, sublayer := range layer.Layers {
		mapped := mapLayer(sublayer, canonicalURL, authorityURL, identifier, append(parentStyleNames, layerStyleNames...))
		l.Layer = append(l.Layer, &mapped)
	}

	return l
}

func getLayerStyles(layer pdoknlv3.Layer, canonicalURL string, parentStyleNames []string) (styles []*wms130.Style) {
	for _, style := range layer.Styles {
		if slices.Contains(parentStyleNames, style.Name) {
			continue
		}

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
		styles = append(styles, &newStyle)
	}
	return
}
