package capabilitiesgenerator

import (
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"testing"

	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

const (
	WFSInput = `global:
    prefix: prefix
    namespace: http://prefix.geonovum.nl
    onlineResourceUrl: http://localhost
    path: /datasetOwner/dataset/theme/wfs/v1_0
    version: v1_0
    additionalSchemaLocations: http://inspire.ec.europa.eu/schemas/inspire_dls/1.0 http://inspire.ec.europa.eu/schemas/inspire_dls/1.0/inspire_dls.xsd
services:
    wfs200:
        filename: /var/www/config/capabilities_wfs_200.xml
        definition:
            serviceIdentification:
                title: some Service title
                abstract: some \"Service\" abstract
                keywords:
                    keyword:
                        - service-keyword-1
                        - service-keyword-2
                        - infoFeatureAccessService
                accessConstraints: http://creativecommons.org/publicdomain/zero/1.0/deed.nl
            serviceProvider:
                providerName: PDOK
            capabilities:
                operationsMetadata:
                    extendedCapabilities:
                        extendedCapabilities:
                            metadataUrl:
                                url: https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=metameta-meta-meta-meta-metametameta
                                mediaType: application/vnd.ogc.csw.GetRecordByIdResponse_xml
                            supportedLanguages:
                                defaultLanguage:
                                    language: dut
                            responseLanguage:
                                language: dut
                            spatialDataSetIdentifier:
                                code: datadata-data-data-data-datadatadata
                featureTypeList:
                    featureType:
                        - name: prefixfeaturetype-1-name
                          title: featuretype-1-title
                          abstract: feature \"1\" abstract
                          keywords:
                            - keyword:
                                - featuretype-1-keyword-1
                                - featuretype-1-keyword-2
                          defaultCrs: urn:ogc:def:crs:EPSG::28992
                          otherCrs:
                            - urn:ogc:def:crs:EPSG::28992
                            - urn:ogc:def:crs:EPSG::25831
                            - urn:ogc:def:crs:EPSG::25832
                            - urn:ogc:def:crs:EPSG::3034
                            - urn:ogc:def:crs:EPSG::3035
                            - urn:ogc:def:crs:EPSG::3857
                            - urn:ogc:def:crs:EPSG::4258
                            - urn:ogc:def:crs:EPSG::4326
                          metadataUrl:
                            href: https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=datadata-data-data-data-datadatadata
                        - name: prefixfeaturetype-2-name
                          title: featuretype-2-title
                          abstract: feature \"2\" abstract
                          keywords:
                            - keyword:
                                - featuretype-2-keyword-1
                                - featuretype-2-keyword-2
                          defaultCrs: urn:ogc:def:crs:EPSG::28992
                          otherCrs:
                            - urn:ogc:def:crs:EPSG::28992
                            - urn:ogc:def:crs:EPSG::25831
                            - urn:ogc:def:crs:EPSG::25832
                            - urn:ogc:def:crs:EPSG::3034
                            - urn:ogc:def:crs:EPSG::3035
                            - urn:ogc:def:crs:EPSG::3857
                            - urn:ogc:def:crs:EPSG::4258
                            - urn:ogc:def:crs:EPSG::4326
                          metadataUrl:
                            href: https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=datadata-data-data-data-datadatadata
`
)

func TestGetInputForWFS(t *testing.T) {
	type args struct {
		WFS       *pdoknlv3.WFS
		ownerInfo *smoothoperatorv1.OwnerInfo
	}
	pdoknlv3.SetHost("http://localhost")
	tests := []struct {
		name      string
		args      args
		wantInput string
		wantErr   bool
	}{
		{
			name: "GetInputForWFS",
			args: args{
				WFS: &pdoknlv3.WFS{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"dataset":         "dataset",
							"dataset-owner":   "datasetOwner",
							"theme":           "theme",
							"service-version": "v1_0",
						},
					},
					Spec: pdoknlv3.WFSSpec{
						Service: pdoknlv3.WFSService{
							URL:               "/datasetOwner/dataset/theme/wfs/v1_0",
							Title:             "some Service title",
							Abstract:          "some \"Service\" abstract",
							Keywords:          []string{"service-keyword-1", "service-keyword-2", "infoFeatureAccessService"},
							AccessConstraints: "http://creativecommons.org/publicdomain/zero/1.0/deed.nl",
							Inspire: &pdoknlv3.Inspire{
								ServiceMetadataURL: pdoknlv3.MetadataURL{
									CSW: &pdoknlv3.Metadata{
										MetadataIdentifier: "metameta-meta-meta-meta-metametameta",
									},
								},
								Language:                 "dut",
								SpatialDatasetIdentifier: "datadata-data-data-data-datadatadata",
							},
							DefaultCrs: "EPSG:28992",
							OtherCrs: []string{
								"EPSG:28992",
								"EPSG:25831",
								"EPSG:25832",
								"EPSG:3034",
								"EPSG:3035",
								"EPSG:3857",
								"EPSG:4258",
								"EPSG:4326",
							},
							FeatureTypes: []pdoknlv3.FeatureType{
								{
									Name:     "featuretype-1-name",
									Title:    "featuretype-1-title",
									Abstract: "feature \"1\" abstract",
									Keywords: []string{"featuretype-1-keyword-1", "featuretype-1-keyword-2"},
									DatasetMetadataURL: pdoknlv3.MetadataURL{
										CSW: &pdoknlv3.Metadata{
											MetadataIdentifier: "datadata-data-data-data-datadatadata",
										},
									},
								},
								{
									Name:     "featuretype-2-name",
									Title:    "featuretype-2-title",
									Abstract: "feature \"2\" abstract",
									Keywords: []string{"featuretype-2-keyword-1", "featuretype-2-keyword-2"},
									DatasetMetadataURL: pdoknlv3.MetadataURL{
										CSW: &pdoknlv3.Metadata{
											MetadataIdentifier: "datadata-data-data-data-datadatadata",
										},
									},
								},
							},
							Prefix: "prefix",
						},
					},
				},
				ownerInfo: &smoothoperatorv1.OwnerInfo{
					Spec: smoothoperatorv1.OwnerInfoSpec{
						NamespaceTemplate: "http://{{prefix}}.geonovum.nl",
						MetadataUrls: smoothoperatorv1.MetadataUrls{
							CSW: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
							},
						},
						WFS: smoothoperatorv1.WFS{
							ServiceProvider: smoothoperatorv1.ServiceProvider{
								ProviderName: smoothoperatorutils.Pointer("PDOK"),
							},
						},
					},
				},
			},
			wantInput: WFSInput,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInput, err := GetInput(tt.args.WFS, tt.args.ownerInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotInput != tt.wantInput {
				t.Errorf("GetInput() gotInput = %v, want %v", gotInput, tt.wantInput)
			}
		})
	}
}

func TestInputForWMS(t *testing.T) {
	var maxSize int32 = 123

	wms := pdoknlv3.WMS{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"dataset":         "dataset",
				"dataset-owner":   "datasetOwner",
				"theme":           "theme",
				"service-version": "v1_0",
			},
		},
		Spec: pdoknlv3.WMSSpec{
			Service: pdoknlv3.WMSService{
				URL:               "/datasetOwner/dataset/theme/wms/v1_0",
				Title:             "some Service title",
				Abstract:          "some \"Service\" abstract",
				Keywords:          []string{"service-keyword-1", "service-keyword-2", "infoFeatureAccessService"},
				AccessConstraints: "http://creativecommons.org/publicdomain/zero/1.0/deed.nl",
				MaxSize:           &maxSize,
				Inspire: &pdoknlv3.Inspire{
					ServiceMetadataURL: pdoknlv3.MetadataURL{
						CSW: &pdoknlv3.Metadata{
							MetadataIdentifier: "metameta-meta-meta-meta-metametameta",
						},
					},
					Language:                 "dut",
					SpatialDatasetIdentifier: "datadata-data-data-data-datadatadata",
				},
				DataEPSG:      "EPSG:28992",
				StylingAssets: nil,
				Mapfile:       nil,
				Layer: pdoknlv3.Layer{
					Name:                "",
					Title:               nil,
					Abstract:            nil,
					Keywords:            nil,
					BoundingBoxes:       nil,
					Visible:             nil,
					Authority:           nil,
					DatasetMetadataURL:  nil,
					MinScaleDenominator: nil,
					MaxScaleDenominator: nil,
					Styles:              nil,
					LabelNoClip:         false,
					Data:                nil,
					Layers:              nil,
				},
			},
		},
	}

	contactPersonPrimary := smoothoperatorv1.ContactPersonPrimary{
		ContactPerson:       asPtr("KlantContactCenter PDOK"),
		ContactOrganization: asPtr("PDOK"),
	}

	ownerInfo := smoothoperatorv1.OwnerInfo{
		Spec: smoothoperatorv1.OwnerInfoSpec{
			NamespaceTemplate: "http://{{prefix}}.geonovum.nl",
			MetadataUrls: smoothoperatorv1.MetadataUrls{
				CSW: smoothoperatorv1.MetadataURL{
					HrefTemplate: "https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
				},
			},
			WMS: smoothoperatorv1.WMS{
				ContactInformation: &smoothoperatorv1.ContactInformation{
					ContactPersonPrimary: &contactPersonPrimary,
					ContactPosition:      asPtr("pointOfContact"),
					ContactAddress: &smoothoperatorv1.ContactAddress{
						AddressType:     asPtr("Work"),
						Address:         nil,
						City:            asPtr("Apeldoorn"),
						StateOrProvince: nil,
						PostCode:        nil,
						Country:         asPtr("The Netherlands"),
					},
					ContactVoiceTelephone:        nil,
					ContactFacsimileTelephone:    nil,
					ContactElectronicMailAddress: asPtr("BeheerPDOK@kadaster.nl"),
				},
			},
		},
	}

	input, err := GetInput(&wms, &ownerInfo)
	assert.NoError(t, err)
	println(input)
}
