package capabilitiesgenerator

import (
	"github.com/google/go-cmp/cmp"
	"github.com/pdok/mapserver-operator/api/v2beta1"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	capabilitiesgenerator "github.com/pdok/ogc-capabilities-generator/pkg/config"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
	yamlv3 "sigs.k8s.io/yaml/goyaml.v3"

	"testing"

	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"

	_ "embed"
)

//go:embed test_data/wfs_input.yaml
var WFSInput string

//go:embed test_data/wms_input.yaml
var WMSInput string

func TestGetInputForWFS(t *testing.T) {
	type args struct {
		WFS       *pdoknlv3.WFS
		ownerInfo *smoothoperatorv1.OwnerInfo
	}
	url, _ := smoothoperatormodel.ParseURL("http://localhost/datasetOwner/dataset/theme/wfs/v1_0")
	pdoknlv3.SetHost("http://localhost")
	accessConstraints, _ := smoothoperatormodel.ParseURL("http://creativecommons.org/publicdomain/zero/1.0/deed.nl")
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
						Service: pdoknlv3.WFSService{BaseService: pdoknlv3.BaseService{
							URL:               smoothoperatormodel.URL{URL: url},
							Prefix:            "prefix",
							Title:             "some Service title",
							Abstract:          "some \"Service\" abstract",
							Keywords:          []string{"service-keyword-1", "service-keyword-2", "infoFeatureAccessService"},
							AccessConstraints: smoothoperatormodel.URL{URL: accessConstraints}},
							Inspire: &pdoknlv3.WFSInspire{Inspire: pdoknlv3.Inspire{
								ServiceMetadataURL: pdoknlv3.MetadataURL{
									CSW: &pdoknlv3.Metadata{
										MetadataIdentifier: "metameta-meta-meta-meta-metametameta",
									},
								},
								Language: "dut"},
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
									DatasetMetadataURL: &pdoknlv3.MetadataURL{
										CSW: &pdoknlv3.Metadata{
											MetadataIdentifier: "datadata-data-data-data-datadatadata",
										},
									},
									Bbox: &pdoknlv3.FeatureBbox{
										WGS84: &smoothoperatormodel.BBox{
											MinX: "-180",
											MaxX: "180",
											MinY: "-90",
											MaxY: "90",
										},
									},
								},
								{
									Name:     "featuretype-2-name",
									Title:    "featuretype-2-title",
									Abstract: "feature \"2\" abstract",
									Keywords: []string{"featuretype-2-keyword-1", "featuretype-2-keyword-2"},
									DatasetMetadataURL: &pdoknlv3.MetadataURL{
										CSW: &pdoknlv3.Metadata{
											MetadataIdentifier: "datadata-data-data-data-datadatadata",
										},
									},
								},
							},
						},
					},
				},
				ownerInfo: &smoothoperatorv1.OwnerInfo{
					Spec: smoothoperatorv1.OwnerInfoSpec{
						NamespaceTemplate: smoothoperatorutils.Pointer("http://{{prefix}}.geonovum.nl"),
						MetadataUrls: &smoothoperatorv1.MetadataUrls{
							CSW: &smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
							},
						},
						WFS: &smoothoperatorv1.WFS{
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

			wantMap := capabilitiesgenerator.Config{}
			gotMap := capabilitiesgenerator.Config{}
			err = yamlv3.Unmarshal([]byte(WFSInput), &wantMap)
			assert.NoError(t, err)
			err = yamlv3.Unmarshal([]byte(gotInput), &gotMap)
			assert.NoError(t, err)

			diff := cmp.Diff(wantMap, gotMap)
			if diff != "" {
				t.Errorf("GetInput() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInputForWMS(t *testing.T) {
	//nolint:misspell
	v2wmsstring := "apiVersion: pdok.nl/v2beta1\nkind: WMS\nmetadata:\n  name: rws-nwbwegen-v1-0\n  labels:\n    dataset-owner: rws\n    dataset: nwbwegen\n    service-version: v1_0\n    service-type: wms\n  annotations:\n    lifecycle-phase: prod\n    service-bundle-id: b39c152b-393b-52f5-a50c-e1ffe904b6fb\nspec:\n  general:\n    datasetOwner: rws\n    dataset: nwbwegen\n    serviceVersion: v1_0\n  kubernetes:\n    healthCheck:\n      boundingbox: 135134.89,457152.55,135416.03,457187.82\n    resources:\n      limits:\n        ephemeralStorage: 1535Mi\n        memory: 4G\n      requests:\n        cpu: 2000m\n        ephemeralStorage: 1535Mi\n        memory: 4G\n  options:\n    automaticCasing: true\n    disableWebserviceProxy: false\n    includeIngress: true\n    validateRequests: true\n  service:\n    title: NWB - Wegen WMS\n    abstract:\n      Dit is de web map service van het Nationaal Wegen Bestand (NWB) - wegen.\n      Deze dataset bevat alleen de wegvakken en hectometerpunten. Het Nationaal Wegen\n      Bestand - Wegen is een digitaal geografisch bestand van alle wegen in Nederland.\n      Opgenomen zijn alle wegen die worden beheerd door wegbeheerders als het Rijk,\n      provincies, gemeenten en waterschappen, echter alleen voor zover deze zijn voorzien\n      van een straatnaam of nummer.\n    authority:\n      name: rws\n      url: https://www.rijkswaterstaat.nl\n    dataEPSG: EPSG:28992\n    extent: -59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961\n    inspire: true\n    keywords:\n      - Vervoersnetwerken\n      - Menselijke gezondheid en veiligheid\n      - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n      - Nationaal\n      - Voertuigen\n      - Verkeer\n      - Wegvakken\n      - Hectometerpunten\n      - HVD\n      - Mobiliteit\n    stylingAssets:\n      configMapRefs:\n        - name: includes\n          keys:\n            - nwb_wegen_hectopunten.symbol\n            - hectopunten.style\n            - wegvakken.style\n      blobKeys:\n        - resources/fonts/liberation-sans.ttf\n    layers:\n      - abstract:\n          Deze laag bevat de wegvakken uit het Nationaal Wegen bestand (NWB)\n          en geeft gedetailleerde informatie per wegvak zoals straatnaam, wegnummer,\n          routenummer, wegbeheerder, huisnummers, enz. weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - wvk_id\n              - wvk_begdat\n              - jte_id_beg\n              - jte_id_end\n              - wegbehsrt\n              - wegnummer\n              - wegdeelltr\n              - hecto_lttr\n              - bst_code\n              - rpe_code\n              - admrichtng\n              - rijrichtng\n              - stt_naam\n              - stt_bron\n              - wpsnaam\n              - gme_id\n              - gme_naam\n              - hnrstrlnks\n              - hnrstrrhts\n              - e_hnr_lnks\n              - e_hnr_rhts\n              - l_hnr_lnks\n              - l_hnr_rhts\n              - begafstand\n              - endafstand\n              - beginkm\n              - eindkm\n              - pos_tv_wol\n              - wegbehcode\n              - wegbehnaam\n              - distrcode\n              - distrnaam\n              - dienstcode\n              - dienstnaam\n              - wegtype\n              - wgtype_oms\n              - routeltr\n              - routenr\n              - routeltr2\n              - routenr2\n              - routeltr3\n              - routenr3\n              - routeltr4\n              - routenr4\n              - wegnr_aw\n              - wegnr_hmp\n              - geobron_id\n              - geobron_nm\n              - bronjaar\n              - openlr\n              - bag_orl\n              - frc\n              - fow\n              - alt_naam\n              - alt_nr\n              - rel_hoogte\n              - st_lengthshape\n            geometryType: MultiLineString\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: wegvakken\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Wegvakken\n        maxScale: 50000.0\n        minScale: 1.0\n        name: wegvakken\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: wegvakken\n            title: NWB - Wegvakken\n            visualization: wegvakken.style\n        title: Wegvakken\n        visible: true\n      - abstract:\n          Deze laag bevat de hectopunten uit het Nationaal Wegen Bestand (NWB)\n          en geeft gedetailleerde informatie per hectopunt zoals hectometrering, afstand,\n          zijde en hectoletter weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - hectomtrng\n              - afstand\n              - wvk_id\n              - wvk_begdat\n              - zijde\n              - hecto_lttr\n            geometryType: MultiPoint\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: hectopunten\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Hectometerpunten\n        maxScale: 50000.0\n        minScale: 1.0\n        name: hectopunten\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: hectopunten\n            title: NWB - Hectopunten\n            visualization: hectopunten.style\n        title: Hectopunten\n        visible: true\n    metadataIdentifier: f2437a92-ddd3-4777-a1bc-fdf4b4a7fcb8\n"
	v2wms := &v2beta1.WMS{}
	err := yaml.Unmarshal([]byte(v2wmsstring), v2wms)
	assert.NoError(t, err)
	pdoknlv3.SetHost("http://localhost")
	var wms pdoknlv3.WMS
	err = v2wms.ToV3(&wms)
	assert.NoError(t, err)

	contactPersonPrimary := smoothoperatorv1.ContactPersonPrimary{
		ContactPerson:       smoothoperatorutils.Pointer("KlantContactCenter PDOK"),
		ContactOrganization: smoothoperatorutils.Pointer("PDOK"),
	}

	ownerInfo := smoothoperatorv1.OwnerInfo{
		Spec: smoothoperatorv1.OwnerInfoSpec{
			NamespaceTemplate: smoothoperatorutils.Pointer("http://{{prefix}}.geonovum.nl"),
			MetadataUrls: &smoothoperatorv1.MetadataUrls{
				CSW: &smoothoperatorv1.MetadataURL{
					HrefTemplate: "https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
				},
			},
			ProviderSite: &smoothoperatorv1.ProviderSite{
				Type: "simple",
				Href: "https://www.pdok.nl",
			},
			WMS: &smoothoperatorv1.WMS{
				ContactInformation: smoothoperatorv1.ContactInformation{
					ContactPersonPrimary: &contactPersonPrimary,
					ContactPosition:      smoothoperatorutils.Pointer("pointOfContact"),
					ContactAddress: &smoothoperatorv1.ContactAddress{
						AddressType:     smoothoperatorutils.Pointer("Work"),
						Address:         nil,
						City:            smoothoperatorutils.Pointer("Apeldoorn"),
						StateOrProvince: nil,
						PostCode:        nil,
						Country:         smoothoperatorutils.Pointer("The Netherlands"),
					},
					ContactVoiceTelephone:        nil,
					ContactFacsimileTelephone:    nil,
					ContactElectronicMailAddress: smoothoperatorutils.Pointer("BeheerPDOK@kadaster.nl"),
				},
			},
		},
	}

	input, err := GetInput(&wms, &ownerInfo)
	assert.NoError(t, err)

	wantMap := capabilitiesgenerator.Config{}
	gotMap := capabilitiesgenerator.Config{}
	err = yamlv3.Unmarshal([]byte(WMSInput), &wantMap)
	assert.NoError(t, err)
	err = yamlv3.Unmarshal([]byte(input), &gotMap)
	assert.NoError(t, err)

	diff := cmp.Diff(wantMap, gotMap)
	assert.Equal(t, diff, "", "%s", diff)
}
