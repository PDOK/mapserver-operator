package capabilitiesgenerator

import (
	"github.com/pdok/mapserver-operator/api/v2beta1"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

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
	v2wmsstring := "apiVersion: pdok.nl/v2beta1\nkind: WMS\nmetadata:\n  name: rws-nwbwegen-v1-0\n  labels:\n    dataset-owner: rws\n    dataset: nwbwegen\n    service-version: v1_0\n    service-type: wms\n  annotations:\n    lifecycle-phase: prod\n    service-bundle-id: b39c152b-393b-52f5-a50c-e1ffe904b6fb\nspec:\n  general:\n    datasetOwner: rws\n    dataset: nwbwegen\n    serviceVersion: v1_0\n  kubernetes:\n    healthCheck:\n      boundingbox: 135134.89,457152.55,135416.03,457187.82\n    resources:\n      limits:\n        ephemeralStorage: 1535Mi\n        memory: 4G\n      requests:\n        cpu: 2000m\n        ephemeralStorage: 1535Mi\n        memory: 4G\n  options:\n    automaticCasing: true\n    disableWebserviceProxy: false\n    includeIngress: true\n    validateRequests: true\n  service:\n    title: NWB - Wegen WMS\n    abstract:\n      Dit is de web map service van het Nationaal Wegen Bestand (NWB) - wegen.\n      Deze dataset bevat alleen de wegvakken en hectometerpunten. Het Nationaal Wegen\n      Bestand - Wegen is een digitaal geografisch bestand van alle wegen in Nederland.\n      Opgenomen zijn alle wegen die worden beheerd door wegbeheerders als het Rijk,\n      provincies, gemeenten en waterschappen, echter alleen voor zover deze zijn voorzien\n      van een straatnaam of nummer.\n    authority:\n      name: rws\n      url: https://www.rijkswaterstaat.nl\n    dataEPSG: EPSG:28992\n    extent: -59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961\n    inspire: true\n    keywords:\n      - Vervoersnetwerken\n      - Menselijke gezondheid en veiligheid\n      - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n      - Nationaal\n      - Voertuigen\n      - Verkeer\n      - Wegvakken\n      - Hectometerpunten\n      - HVD\n      - Mobiliteit\n    stylingAssets:\n      configMapRefs:\n        - name: includes\n          keys:\n            - nwb_wegen_hectopunten.symbol\n            - hectopunten.style\n            - wegvakken.style\n      blobKeys:\n        - resources/fonts/liberation-sans.ttf\n    layers:\n      - abstract:\n          Deze laag bevat de wegvakken uit het Nationaal Wegen bestand (NWB)\n          en geeft gedetailleerde informatie per wegvak zoals straatnaam, wegnummer,\n          routenummer, wegbeheerder, huisnummers, enz. weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - wvk_id\n              - wvk_begdat\n              - jte_id_beg\n              - jte_id_end\n              - wegbehsrt\n              - wegnummer\n              - wegdeelltr\n              - hecto_lttr\n              - bst_code\n              - rpe_code\n              - admrichtng\n              - rijrichtng\n              - stt_naam\n              - stt_bron\n              - wpsnaam\n              - gme_id\n              - gme_naam\n              - hnrstrlnks\n              - hnrstrrhts\n              - e_hnr_lnks\n              - e_hnr_rhts\n              - l_hnr_lnks\n              - l_hnr_rhts\n              - begafstand\n              - endafstand\n              - beginkm\n              - eindkm\n              - pos_tv_wol\n              - wegbehcode\n              - wegbehnaam\n              - distrcode\n              - distrnaam\n              - dienstcode\n              - dienstnaam\n              - wegtype\n              - wgtype_oms\n              - routeltr\n              - routenr\n              - routeltr2\n              - routenr2\n              - routeltr3\n              - routenr3\n              - routeltr4\n              - routenr4\n              - wegnr_aw\n              - wegnr_hmp\n              - geobron_id\n              - geobron_nm\n              - bronjaar\n              - openlr\n              - bag_orl\n              - frc\n              - fow\n              - alt_naam\n              - alt_nr\n              - rel_hoogte\n              - st_lengthshape\n            geometryType: MultiLineString\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: wegvakken\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Wegvakken\n        maxScale: 50000.0\n        minScale: 1.0\n        name: wegvakken\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: wegvakken\n            title: NWB - Wegvakken\n            visualization: wegvakken.style\n        title: Wegvakken\n        visible: true\n      - abstract:\n          Deze laag bevat de hectopunten uit het Nationaal Wegen Bestand (NWB)\n          en geeft gedetailleerde informatie per hectopunt zoals hectometrering, afstand,\n          zijde en hectoletter weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - hectomtrng\n              - afstand\n              - wvk_id\n              - wvk_begdat\n              - zijde\n              - hecto_lttr\n            geometryType: MultiPoint\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: hectopunten\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Hectometerpunten\n        maxScale: 50000.0\n        minScale: 1.0\n        name: hectopunten\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: hectopunten\n            title: NWB - Hectopunten\n            visualization: hectopunten.style\n        title: Hectopunten\n        visible: true\n    metadataIdentifier: f2437a92-ddd3-4777-a1bc-fdf4b4a7fcb8\n"
	var v2wms v2beta1.WMS
	err := yaml.Unmarshal([]byte(v2wmsstring), &v2wms)
	assert.NoError(t, err)
	var wms pdoknlv3.WMS
	v2beta1.V3HubFromV2(&v2wms, &wms)

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
