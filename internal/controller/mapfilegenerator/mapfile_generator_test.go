package mapfilegenerator

import (
	"encoding/json"
	"github.com/pdok/mapserver-operator/api/v2beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
	"testing"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	shared_model "github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WFSConfig = `{
    "service_title": "some Service title",
    "service_abstract": "some \\\"Service\\\" abstract",
    "service_keywords": "service-keyword-1,service-keyword-2,infoFeatureAccessService",
    "service_extent": "0.0 2.0 1.0 3.0",
    "service_wfs_maxfeatures": "1000",
    "service_namespace_prefix": "prefix",
    "service_namespace_uri": "http://prefix.geonovum.nl",
    "service_onlineresource": "http://localhost",
    "service_path": "/datasetOwner/dataset/theme/wfs/v1_0",
    "service_metadata_id": "metameta-meta-meta-meta-metametameta",
    "automatic_casing": true,
    "data_epsg": "EPSG:28992",
    "epsg_list": [
        "EPSG:28992",
        "EPSG:25831",
        "EPSG:25832",
        "EPSG:3034",
        "EPSG:3035",
        "EPSG:3857",
        "EPSG:4258",
        "EPSG:4326"
    ],
    "layers": [
        {
            "name": "featuretype-1-name",
            "title": "featuretype-1-title",
            "abstract": "feature \\\"1\\\" abstract",
            "keywords": "featuretype-1-keyword-1,featuretype-1-keyword-2",
            "layer_extent": "0.0 2.0 1.0 3.0",
            "dataset_metadata_id": "datadata-data-data-data-datadatadata",
            "columns": [
                {
                    "name": "fuuid"
                },
                {
                    "name": "featuretype-1-column-1"
                },
                {
                    "name": "featuretype-1-column-2"
                }
            ],
            "geometry_type": "Point",
            "gpkg_path": "/srv/data/gpkg/file-1.gpkg",
            "tablename": "featuretype-1"
        },
        {
            "name": "featuretype-2-name",
            "title": "featuretype-2-title",
            "abstract": "feature \\\"2\\\" abstract",
            "keywords": "featuretype-2-keyword-1,featuretype-2-keyword-2",
            "layer_extent": "0.0 2.0 1.0 3.0",
            "dataset_metadata_id": "datadata-data-data-data-datadatadata",
            "columns": [
                {
                    "name": "fuuid"
                },
                {
                    "name": "featuretype-2-column-1",
                    "alias": "alias_featuretype-2-column-1"
                },
                {
                    "name": "featuretype-2-column-2"
                }
            ],
            "geometry_type": "MultiLine",
            "tablename": "featuretype-2",
            "postgis": true
        }
    ]
}`
	WMSConfig = `{"service_title": "NWB - Wegen WMS", "service_abstract": "Dit is de web map service van het Nationaal Wegen Bestand (NWB) - wegen. Deze dataset bevat alleen de wegvakken en hectometerpunten. Het Nationaal Wegen Bestand - Wegen is een digitaal geografisch bestand van alle wegen in Nederland. Opgenomen zijn alle wegen die worden beheerd door wegbeheerders als het Rijk, provincies, gemeenten en waterschappen, echter alleen voor zover deze zijn voorzien van een straatnaam of nummer.", "service_keywords": "Vervoersnetwerken,Menselijke gezondheid en veiligheid,Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai),Nationaal,Voertuigen,Verkeer,Wegvakken,Hectometerpunten,HVD,Mobiliteit,infoMapAccessService", "service_accessconstraints": "https://creativecommons.org/publicdomain/zero/1.0/deed.nl", "service_extent": "-59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961", "maxSize": "4000", "service_namespace_prefix": "nwbwegen", "service_namespace_uri": "http://nwbwegen.geonovum.nl", "service_onlineresource": "https://service.pdok.nl", "service_path": "/rws/nwbwegen/wms/v1_0", "service_metadata_id": "f2437a92-ddd3-4777-a1bc-fdf4b4a7fcb8", "dataset_owner": "rws", "authority_url": "https://www.rijkswaterstaat.nl", "automatic_casing": true, "data_epsg": "EPSG:28992", "epsg_list": ["EPSG:28992", "EPSG:25831", "EPSG:25832", "EPSG:3034", "EPSG:3035", "EPSG:3857", "EPSG:4258", "EPSG:4326", "CRS:84"], "templates": "/srv/data/config/templates", "fonts": "/srv/data/config/fonts", "outputformat_jpg": "jpg", "outputformat_png8": "png", "symbols": ["/styling/nwb_wegen_hectopunten.symbol"], "group_layers": [], "layers": [{"name": "wegvakken", "gpkg_path": "/srv/data/gpkg/nwb_wegen.gpkg", "tablename": "wegvakken", "geometry_type": "MultiLineString", "columns": [{"name": "fuuid"}, {"name": "objectid"}, {"name": "wvk_id"}, {"name": "wvk_begdat"}, {"name": "jte_id_beg"}, {"name": "jte_id_end"}, {"name": "wegbehsrt"}, {"name": "wegnummer"}, {"name": "wegdeelltr"}, {"name": "hecto_lttr"}, {"name": "bst_code"}, {"name": "rpe_code"}, {"name": "admrichtng"}, {"name": "rijrichtng"}, {"name": "stt_naam"}, {"name": "stt_bron"}, {"name": "wpsnaam"}, {"name": "gme_id"}, {"name": "gme_naam"}, {"name": "hnrstrlnks"}, {"name": "hnrstrrhts"}, {"name": "e_hnr_lnks"}, {"name": "e_hnr_rhts"}, {"name": "l_hnr_lnks"}, {"name": "l_hnr_rhts"}, {"name": "begafstand"}, {"name": "endafstand"}, {"name": "beginkm"}, {"name": "eindkm"}, {"name": "pos_tv_wol"}, {"name": "wegbehcode"}, {"name": "wegbehnaam"}, {"name": "distrcode"}, {"name": "distrnaam"}, {"name": "dienstcode"}, {"name": "dienstnaam"}, {"name": "wegtype"}, {"name": "wgtype_oms"}, {"name": "routeltr"}, {"name": "routenr"}, {"name": "routeltr2"}, {"name": "routenr2"}, {"name": "routeltr3"}, {"name": "routenr3"}, {"name": "routeltr4"}, {"name": "routenr4"}, {"name": "wegnr_aw"}, {"name": "wegnr_hmp"}, {"name": "geobron_id"}, {"name": "geobron_nm"}, {"name": "bronjaar"}, {"name": "openlr"}, {"name": "bag_orl"}, {"name": "frc"}, {"name": "fow"}, {"name": "alt_naam"}, {"name": "alt_nr"}, {"name": "rel_hoogte"}, {"name": "st_lengthshape"}], "title": "Wegvakken", "abstract": "Deze laag bevat de wegvakken uit het Nationaal Wegen bestand (NWB) en geeft gedetailleerde informatie per wegvak zoals straatnaam, wegnummer, routenummer, wegbeheerder, huisnummers, enz. weer.", "keywords": "Vervoersnetwerken,Menselijke gezondheid en veiligheid,Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai),Nationaal,Voertuigen,Verkeer,Wegvakken", "dataset_metadata_id": "a9b7026e-0a81-4813-93bd-ba49e6f28502", "dataset_source_id": "8f0497f0-dbd7-4bee-b85a-5fdec484a7ff", "layer_extent": "-59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961", "minscale": "1", "maxscale": "50000", "styles": [{"title": "NWB - Wegvakken", "path": "/styling/wegvakken.style"}]}, {"name": "hectopunten", "gpkg_path": "/srv/data/gpkg/nwb_wegen.gpkg", "tablename": "hectopunten", "geometry_type": "MultiPoint", "columns": [{"name": "fuuid"}, {"name": "objectid"}, {"name": "hectomtrng"}, {"name": "afstand"}, {"name": "wvk_id"}, {"name": "wvk_begdat"}, {"name": "zijde"}, {"name": "hecto_lttr"}], "title": "Hectopunten", "abstract": "Deze laag bevat de hectopunten uit het Nationaal Wegen Bestand (NWB) en geeft gedetailleerde informatie per hectopunt zoals hectometrering, afstand, zijde en hectoletter weer.", "keywords": "Vervoersnetwerken,Menselijke gezondheid en veiligheid,Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai),Nationaal,Voertuigen,Verkeer,Hectometerpunten", "dataset_metadata_id": "a9b7026e-0a81-4813-93bd-ba49e6f28502", "dataset_source_id": "8f0497f0-dbd7-4bee-b85a-5fdec484a7ff", "layer_extent": "-59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961", "minscale": "1", "maxscale": "50000", "styles": [{"title": "NWB - Hectopunten", "path": "/styling/hectopunten.style"}]}]}`
)

func TestGetConfigForWFS(t *testing.T) {
	type args struct {
		WFS       *pdoknlv3.WFS
		ownerInfo *smoothoperatorv1.OwnerInfo
	}
	pdoknlv3.SetHost("http://localhost")
	tests := []struct {
		name       string
		args       args
		wantConfig string
		wantErr    bool
	}{
		{
			name: "GetConfig for WFS",
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
						Options: pdoknlv3.Options{
							AutomaticCasing: true,
						},
						Service: pdoknlv3.WFSService{
							Title:             "some Service title",
							Abstract:          "some \"Service\" abstract",
							Keywords:          []string{"service-keyword-1", "service-keyword-2", "infoFeatureAccessService"},
							AccessConstraints: "http://creativecommons.org/publicdomain/zero/1.0/deed.nl",
							Bbox: &pdoknlv3.Bbox{
								DefaultCRS: shared_model.BBox{
									MinX: "0.0",
									MaxX: "1.0",
									MinY: "2.0",
									MaxY: "3.0",
								},
							},
							Inspire: &pdoknlv3.Inspire{
								ServiceMetadataURL: pdoknlv3.MetadataURL{
									CSW: &pdoknlv3.Metadata{
										MetadataIdentifier: "metameta-meta-meta-meta-metametameta",
									},
								},
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
									Bbox: &pdoknlv3.FeatureBbox{
										DefaultCRS: shared_model.BBox{
											MinX: "0.0",
											MaxX: "1.0",
											MinY: "2.0",
											MaxY: "3.0",
										},
									},
									Data: pdoknlv3.Data{
										Gpkg: &pdoknlv3.Gpkg{
											TableName:    "featuretype-1",
											GeometryType: "Point",
											BlobKey:      "public/testme/gpkg/file-1.gpkg",
											Columns: []pdoknlv3.Column{
												{Name: "featuretype-1-column-1"},
												{Name: "featuretype-1-column-2"},
											},
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
									Bbox: &pdoknlv3.FeatureBbox{
										DefaultCRS: shared_model.BBox{
											MinX: "0.0",
											MaxX: "1.0",
											MinY: "2.0",
											MaxY: "3.0",
										},
									},
									Data: pdoknlv3.Data{
										Postgis: &pdoknlv3.Postgis{
											TableName:    "featuretype-2",
											GeometryType: "MultiLine",
											Columns: []pdoknlv3.Column{
												{Name: "featuretype-2-column-1", Alias: smoothoperatorutils.Pointer("alias_featuretype-2-column-1")},
												{Name: "featuretype-2-column-2"},
											},
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
					},
				},
			},
			wantConfig: WFSConfig,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConfig, err := GetConfig(tt.args.WFS, tt.args.ownerInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			require.JSONEqf(t, tt.wantConfig, gotConfig, "GetConfig() got %v, want %v", gotConfig, tt.wantConfig)
		})
	}
}

func TestGetConfigForWMS(t *testing.T) {
	ownerInfo := &smoothoperatorv1.OwnerInfo{
		Spec: smoothoperatorv1.OwnerInfoSpec{
			NamespaceTemplate: "http://{{prefix}}.geonovum.nl",
		},
	}

	v2wmsstring := "apiVersion: pdok.nl/v2beta1\nkind: WMS\nmetadata:\n  name: rws-nwbwegen-v1-0\n  labels:\n    dataset-owner: rws\n    dataset: nwbwegen\n    service-version: v1_0\n    service-type: wms\n  annotations:\n    lifecycle-phase: prod\n    service-bundle-id: b39c152b-393b-52f5-a50c-e1ffe904b6fb\nspec:\n  general:\n    datasetOwner: rws\n    dataset: nwbwegen\n    serviceVersion: v1_0\n  kubernetes:\n    healthCheck:\n      boundingbox: 135134.89,457152.55,135416.03,457187.82\n    resources:\n      limits:\n        ephemeralStorage: 1535Mi\n        memory: 4G\n      requests:\n        cpu: 2000m\n        ephemeralStorage: 1535Mi\n        memory: 4G\n  options:\n    automaticCasing: true\n    disableWebserviceProxy: false\n    includeIngress: true\n    validateRequests: true\n  service:\n    title: NWB - Wegen WMS\n    abstract:\n      Dit is de web map service van het Nationaal Wegen Bestand (NWB) - wegen.\n      Deze dataset bevat alleen de wegvakken en hectometerpunten. Het Nationaal Wegen\n      Bestand - Wegen is een digitaal geografisch bestand van alle wegen in Nederland.\n      Opgenomen zijn alle wegen die worden beheerd door wegbeheerders als het Rijk,\n      provincies, gemeenten en waterschappen, echter alleen voor zover deze zijn voorzien\n      van een straatnaam of nummer.\n    authority:\n      name: rws\n      url: https://www.rijkswaterstaat.nl\n    dataEPSG: EPSG:28992\n    extent: -59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961\n    inspire: true\n    keywords:\n      - Vervoersnetwerken\n      - Menselijke gezondheid en veiligheid\n      - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n      - Nationaal\n      - Voertuigen\n      - Verkeer\n      - Wegvakken\n      - Hectometerpunten\n      - HVD\n      - Mobiliteit\n    stylingAssets:\n      configMapRefs:\n        - name: includes\n          keys:\n            - nwb_wegen_hectopunten.symbol\n            - hectopunten.style\n            - wegvakken.style\n      blobKeys:\n        - resources/fonts/liberation-sans.ttf\n    layers:\n      - abstract:\n          Deze laag bevat de wegvakken uit het Nationaal Wegen bestand (NWB)\n          en geeft gedetailleerde informatie per wegvak zoals straatnaam, wegnummer,\n          routenummer, wegbeheerder, huisnummers, enz. weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - wvk_id\n              - wvk_begdat\n              - jte_id_beg\n              - jte_id_end\n              - wegbehsrt\n              - wegnummer\n              - wegdeelltr\n              - hecto_lttr\n              - bst_code\n              - rpe_code\n              - admrichtng\n              - rijrichtng\n              - stt_naam\n              - stt_bron\n              - wpsnaam\n              - gme_id\n              - gme_naam\n              - hnrstrlnks\n              - hnrstrrhts\n              - e_hnr_lnks\n              - e_hnr_rhts\n              - l_hnr_lnks\n              - l_hnr_rhts\n              - begafstand\n              - endafstand\n              - beginkm\n              - eindkm\n              - pos_tv_wol\n              - wegbehcode\n              - wegbehnaam\n              - distrcode\n              - distrnaam\n              - dienstcode\n              - dienstnaam\n              - wegtype\n              - wgtype_oms\n              - routeltr\n              - routenr\n              - routeltr2\n              - routenr2\n              - routeltr3\n              - routenr3\n              - routeltr4\n              - routenr4\n              - wegnr_aw\n              - wegnr_hmp\n              - geobron_id\n              - geobron_nm\n              - bronjaar\n              - openlr\n              - bag_orl\n              - frc\n              - fow\n              - alt_naam\n              - alt_nr\n              - rel_hoogte\n              - st_lengthshape\n            geometryType: MultiLineString\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: wegvakken\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Wegvakken\n        maxScale: 50000.0\n        minScale: 1.0\n        name: wegvakken\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: wegvakken\n            title: NWB - Wegvakken\n            visualization: wegvakken.style\n        title: Wegvakken\n        visible: true\n      - abstract:\n          Deze laag bevat de hectopunten uit het Nationaal Wegen Bestand (NWB)\n          en geeft gedetailleerde informatie per hectopunt zoals hectometrering, afstand,\n          zijde en hectoletter weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - hectomtrng\n              - afstand\n              - wvk_id\n              - wvk_begdat\n              - zijde\n              - hecto_lttr\n            geometryType: MultiPoint\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: hectopunten\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Hectometerpunten\n        maxScale: 50000.0\n        minScale: 1.0\n        name: hectopunten\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: hectopunten\n            title: NWB - Hectopunten\n            visualization: hectopunten.style\n        title: Hectopunten\n        visible: true\n    metadataIdentifier: f2437a92-ddd3-4777-a1bc-fdf4b4a7fcb8\n"
	var v2wms v2beta1.WMS
	err := yaml.Unmarshal([]byte(v2wmsstring), &v2wms)
	assert.NoError(t, err)
	var wms pdoknlv3.WMS
	v2beta1.V3HubFromV2(&v2wms, &wms)

	pdoknlv3.SetHost("http://localhost")

	inputStruct, err := MapWMSToMapfileGeneratorInput(&wms, ownerInfo)
	assert.NoError(t, err)
	expected := WMSInput{}
	err = json.Unmarshal([]byte(WMSConfig), &expected)
	assert.NoError(t, err)

	assert.Equal(t, expected, inputStruct)
}
