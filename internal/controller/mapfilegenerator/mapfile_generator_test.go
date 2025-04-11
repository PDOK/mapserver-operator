package mapfilegenerator

import (
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
	WMSConfig = ``
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

	v2wmsstring := "apiVersion: pdok.nl/v2beta1\nkind: WMS\nmetadata:\n  name: kadaster-kadastralekaart\n  labels:\n    dataset-owner: kadaster\n    dataset: kadastralekaart\n    service-version: v5_0\n    service-type: wms\nspec:\n  general:\n    datasetOwner: kadaster\n    dataset: kadastralekaart\n    serviceVersion: v5_0\n  kubernetes:\n    healthCheck:\n      querystring: language=dut&SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&BBOX=193882.0336615453998,470528.1693874415942,193922.4213813782844,470564.250484353397&CRS=EPSG:28992&WIDTH=769&HEIGHT=687&LAYERS=OpenbareRuimteNaam,Bebouwing,Perceel,KadastraleGrens&FORMAT=image/png&DPI=96&MAP_RESOLUTION=96&FORMAT_OPTIONS=dpi:96&TRANSPARENT=TRUE\n      mimetype: image/png\n    resources:\n      limits:\n        memory: \"100M\"\n        ephemeralStorage: \"200M\"\n      requests:\n        cpu: \"500\"\n        memory: \"100M\"\n        ephemeralStorage: \"100M\"\n  options:\n    automaticCasing: true\n    disableWebserviceProxy: false\n    includeIngress: true\n    validateRequests: true\n    rewriteGroupToDataLayers: true\n  service:\n    inspire: false\n    title: Kadastrale Kaart (WMS)\n    abstract: Overzicht van de ligging van de kadastrale percelen in Nederland. Fungeert als schakel tussen terrein en registratie, vervult voor externe gebruiker vaak een referentiefunctie, een ondergrond ten opzichte waarvan de gebruiker eigen informatie kan vastleggen en presenteren.\n    keywords:\n      - Kadaster\n      - Kadastrale percelen\n      - Kadastrale grens\n      - Kadastrale kaart\n      - Bebouwing\n      - Nummeraanduidingreeks\n      - Openbare ruimte naam\n      - Perceel\n      - Grens\n      - Kwaliteit\n      - Kwaliteitslabels\n      - HVD\n      - Geospatiale data\n    metadataIdentifier: 97cf6a64-9cfc-4ce6-9741-2db44fd27fca\n    authority:\n      name: kadaster\n      url: https://www.kadaster.nl\n    dataEPSG: EPSG:28992\n    resolution: 91\n    defResolution: 91\n    extent: \"-25000 250000 280000 860000\"\n    maxSize: 10000\n    stylingAssets:\n      configMapRefs:\n        - name: ${INCLUDES}\n      blobKeys:\n        - ${BLOBS_RESOURCES_BUCKET}/fonts/liberation-sans.ttf\n        - ${BLOBS_RESOURCES_BUCKET}/fonts/liberation-sans-italic.ttf\n    layers:\n      - name: Kadastralekaart\n        title: KadastraleKaartv5\n        abstract: Overzicht van de ligging van de kadastrale percelen in Nederland. Fungeert als schakel tussen terrein en registratie, vervult voor externe gebruiker vaak een referentiefunctie, een ondergrond ten opzichte waarvan de gebruiker eigen informatie kan vastleggen en presenteren.\n        maxScale: 6001\n        keywords:\n          - Kadaster\n          - Kadastrale percelen\n          - Kadastrale grens\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n      - name: Bebouwing\n        visible: true\n        group: Kadastralekaart\n        title: Bebouwing\n        abstract: De laag Bebouwing is een selectie op panden van de BGT.\n        keywords:\n          - Bebouwing\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard:bebouwing\n            title: Standaardvisualisatie Bebouwing\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n          - name: kwaliteit:bebouwing\n            title: Kwaliteitsvisualisatie Bebouwing\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n          - name: print:bebouwing\n            title: Printvisualisatie Bebouwing\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n      - name: Bebouwingvlak\n        visible: true\n        group: Bebouwing\n        title: Bebouwingvlak\n        abstract: De laag Bebouwing is een selectie op panden van de BGT.\n        keywords:\n          - Bebouwing\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: bebouwing.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: bebouwing_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: bebouwing_print.style\n          - name: standaard:bebouwing\n            title: Standaardvisualisatie Bebouwing\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: bebouwing.group.style\n          - name: kwaliteit:bebouwing\n            title: Kwaliteitsvisualisatie Bebouwing\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: bebouwing_kwaliteit.group.style\n          - name: print:bebouwing\n            title: Printvisualisatie Bebouwing\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: bebouwing_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/pand.gpkg\n            table: pand\n            geometryType: Polygon\n            columns:\n              - object_begin_tijd\n              - lv_publicatiedatum\n              - relatieve_hoogteligging\n              - in_onderzoek\n              - tijdstip_registratie\n              - identificatie_namespace\n              - identificatie_lokaal_id\n              - bronhouder\n              - bgt_status\n              - plus_status\n              - identificatie_bag_pnd\n            aliases:\n              lv_publicatiedatum: LV-publicatiedatum\n              identificatie_lokaal_id: identificatieLokaalID\n              identificatie_bag_pnd: identificatieBAGPND\n              bgt_status: bgt-status\n              plus_status: plus-status\n      - name: Nummeraanduidingreeks\n        visible: true\n        group: Bebouwing\n        title: Nummeraanduidingreeks\n        abstract: De laag Bebouwing is een selectie op panden van de BGT.\n        keywords:\n          - Nummeraanduidingreeks\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 2001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaarvisualisatie van de nummeraanduidingreeks.\n            visualization: nummeraanduidingreeks.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: nummeraanduidingreeks_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: nummeraanduidingreeks_print.style\n          - name: standaard:bebouwing\n            title: Standaardvisualisatie Bebouwing\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: nummeraanduidingreeks.group.style\n          - name: kwaliteit:bebouwing\n            title: Kwaliteitsvisualisatie Bebouwing\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: nummeraanduidingreeks_kwaliteit.group.style\n          - name: print:bebouwing\n            title: Printvisualisatie Bebouwing\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: nummeraanduidingreeks_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/pand_nummeraanduiding.gpkg\n            table: pand_nummeraanduiding\n            geometryType: Point\n            columns:\n              - bebouwing_id\n              - hoek\n              - tekst\n              - bag_vbo_laagste_huisnummer\n              - bag_vbo_hoogste_huisnummer\n              - hoek\n            aliases:\n              bebouwing_id: bebouwingID\n              bag_vbo_laagste_huisnummer: identificatie_BAGVBOLaagsteHuisnummer\n              bag_vbo_hoogste_huisnummer: identificatie_BAGVBOHoogsteHuisnummer\n      - name: OpenbareRuimteNaam\n        visible: true\n        group: Kadastralekaart\n        title: OpenbareRuimteNaam\n        abstract: De laag Openbareruimtenaam is een selectie op de openbare ruimte labels van de BGT met een bgt-status \"bestaand\" die een classificatie (openbareruimtetype) Weg en Water hebben.\n        keywords:\n          - Openbare ruimte naam\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 2001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: openbareruimtenaam.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: openbareruimtenaam_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: openbareruimtenaam_print.style\n          - name: standaard:openbareruimtenaam\n            title: Standaardvisualisatie OpenbareRuimteNaam\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: openbareruimtenaam.group.style\n          - name: kwaliteit:openbareruimtenaam\n            title: Kwaliteitsvisualisatie OpenbareRuimteNaam\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: openbareruimtenaam_kwaliteit.group.style\n          - name: print:openbareruimtenaam\n            title: Printvisualisatie OpenbareRuimteNaam\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: openbareruimtenaam_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/openbareruimtelabel.gpkg\n            table: openbareruimtelabel\n            geometryType: Point\n            columns:\n              - object_begin_tijd\n              - lv_publicatiedatum\n              - relatieve_hoogteligging\n              - in_onderzoek\n              - tijdstip_registratie\n              - identificatie_namespace\n              - identificatie_lokaal_id\n              - bronhouder\n              - bgt_status\n              - plus_status\n              - identificatie_bag_opr\n              - tekst\n              - hoek\n              - openbare_ruimte_type\n            aliases:\n              lv_publicatiedatum: LV-publicatiedatum\n              identificatie_lokaal_id: identificatieLokaalID\n              identificatie_bag_opr: identificatieBAGOPR\n              bgt_status: bgt-status\n              plus_status: plus-status\n      - name: Perceel\n        visible: true\n        group: Kadastralekaart\n        title: Perceel\n        abstract: Een perceel is een stuk grond waarvan het Kadaster de grenzen heeft gemeten of gaat meten en dat bij het Kadaster een eigen nummer heeft. Een perceel is een begrensd deel van het Nederlands grondgebied dat kadastraal geïdentificeerd is en met kadastrale grenzen begrensd is.\n        keywords:\n          - Perceel\n          - Kadastrale percelen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard:perceel\n            title: Standaardvisualisatie Perceel\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n          - name: kwaliteit:perceel\n            title: Kwaliteitsvisualisatie Perceel\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n          - name: print:perceel\n            title: Printvisualisatie Perceel\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n      - name: Perceelvlak\n        visible: true\n        group: Perceel\n        title: Perceelvlak\n        abstract: Een perceel is een stuk grond waarvan het Kadaster de grenzen heeft gemeten of gaat meten en dat bij het Kadaster een eigen nummer heeft. Een perceel is een begrensd deel van het Nederlands grondgebied dat kadastraal geïdentificeerd is en met kadastrale grenzen begrensd is.\n        keywords:\n          - Kadastrale percelen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: perceelvlak.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: perceelvlak_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: perceelvlak_print.style\n          - name: standaard:perceel\n            title: Standaardvisualisatie Perceel\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: perceelvlak.group.style\n          - name: kwaliteit:perceel\n            title: Kwaliteitsvisualisatie Perceel\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: perceelvlak_kwaliteit.group.style\n          - name: print:perceel\n            title: Printvisualisatie Perceel\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: perceelvlak_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/perceel.gpkg\n            table: perceel\n            geometryType: Polygon\n            columns:\n              - identificatie_namespace\n              - identificatie_lokaal_id\n              - begin_geldigheid\n              - tijdstip_registratie\n              - volgnummer\n              - status_historie_code\n              - status_historie_waarde\n              - kadastrale_gemeente_code\n              - kadastrale_gemeente_waarde\n              - sectie\n              - akr_kadastrale_gemeente_code_code\n              - akr_kadastrale_gemeente_code_waarde\n              - kadastrale_grootte_waarde\n              - soort_grootte_code\n              - soort_grootte_waarde\n              - perceelnummer\n              - perceelnummer_rotatie\n              - perceelnummer_verschuiving_delta_x\n              - perceelnummer_verschuiving_delta_y\n              - perceelnummer_plaatscoordinaat_x\n              - perceelnummer_plaatscoordinaat_y\n            aliases:\n              identificatie_lokaal_id: identificatieLokaalID\n              akr_kadastrale_gemeente_code_code: AKRKadastraleGemeenteCodeCode\n              akr_kadastrale_gemeente_code_waarde: AKRKadastraleGemeenteCodeWaarde\n      - name: Label\n        visible: true\n        group: Perceel\n        title: Label\n        abstract: Een perceel is een stuk grond waarvan het Kadaster de grenzen heeft gemeten of gaat meten en dat bij het Kadaster een eigen nummer heeft. Een perceel is een begrensd deel van het Nederlands grondgebied dat kadastraal geïdentificeerd is en met kadastrale grenzen begrensd is.\n        keywords:\n          - Kadastrale percelen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaarvisualisatie van het label.\n            visualization: label.style\n          - name: standaard:perceel\n            title: Standaardvisualisatie Perceel\n            abstract: Standaarvisualisatie van het label.\n            visualization: label.group.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: label_kwaliteit.style\n          - name: kwaliteit:perceel\n            title: Kwaliteitsvisualisatie Perceel\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: label_kwaliteit.group.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: label_print.style\n          - name: print:perceel\n            title: Printvisualisatie Perceel\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: label_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/perceel_label.gpkg\n            table: perceel_label\n            geometryType: Point\n            columns:\n              - perceel_id\n              - perceelnummer\n              - rotatie\n              - verschuiving_delta_x\n              - verschuiving_delta_y\n            aliases:\n              perceel_id: perceelID\n      - name: Bijpijling\n        visible: true\n        group: Perceel\n        title: Bijpijling\n        abstract: Een perceel is een stuk grond waarvan het Kadaster de grenzen heeft gemeten of gaat meten en dat bij het Kadaster een eigen nummer heeft. Een perceel is een begrensd deel van het Nederlands grondgebied dat kadastraal geïdentificeerd is en met kadastrale grenzen begrensd is.\n        keywords:\n          - Kadastrale percelen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: bijpijling.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: bijpijling_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: bijpijling_print.style\n          - name: standaard:perceel\n            title: Standaardvisualisatie Perceel\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: bijpijling.group.style\n          - name: kwaliteit:perceel\n            title: Kwaliteitsvisualisatie Perceel\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: bijpijling_kwaliteit.group.style\n          - name: print:perceel\n            title: Printvisualisatie Perceel\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: bijpijling_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/perceel_bijpijling.gpkg\n            table: perceel_bijpijling\n            geometryType: LineString\n            columns:\n              - perceel_id\n            aliases:\n              perceel_id: perceelID\n      - name: KadastraleGrens\n        visible: true\n        group: Kadastralekaart\n        title: KadastraleGrens\n        abstract: Een Kadastrale Grens is de weergave van een grens op de kadastrale kaart die door de dienst van het Kadaster tussen percelen (voorlopig) vastgesteld wordt, op basis van inlichtingen van belanghebbenden en met  gebruikmaking van de aan de kadastrale kaart ten grondslag liggende bescheiden die in elk geval de landmeetkundige gegevens bevatten van hetgeen op die kaart wordt weergegeven.\n        keywords:\n          - Grens\n          - Kadastrale grenzen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: kadastralegrens.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: kadastralegrens_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: kadastralegrens_print.style\n          - name: standaard:kadastralegrens\n            title: Standaardvisualisatie KadastraleGrens\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: kadastralegrens.group.style\n          - name: kwaliteit:kadastralegrens\n            title: Kwaliteitsvisualisatie KadastraleGrens\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: kadastralegrens_kwaliteit.group.style\n          - name: print:kadastralegrens\n            title: Printvisualisatie KadastraleGrens\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: kadastralegrens_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/kadastrale_grens.gpkg\n            table: kadastrale_grens\n            geometryType: LineString\n            columns:\n              - begin_geldigheid\n              - tijdstip_registratie\n              - volgnummer\n              - status_historie_code\n              - status_historie_waarde\n              - identificatie_namespace\n              - identificatie_lokaal_id\n              - type_grens_code\n              - type_grens_waarde\n              - classificatie_kwaliteit_code\n              - classificatie_kwaliteit_waarde\n              - perceel_links_identificatie_namespace\n              - perceel_links_identificatie_lokaal_id\n              - perceel_rechts_identificatie_namespace\n              - perceel_rechts_identificatie_lokaal_id\n            aliases:\n              identificatie_lokaal_id: identificatieLokaalID\n              perceel_links_identificatie_lokaal_id: perceelLinksIdentificatieLokaalID\n              perceel_rechts_identificatie_lokaal_id: perceelRechtsIdentificatieLokaalID\n              classificatie_kwaliteit_code: ClassificatieKwaliteitCode\n              classificatie_kwaliteit_waarde: ClassificatieKwaliteitWaarde\n"
	var v2wms v2beta1.WMS
	err := yaml.Unmarshal([]byte(v2wmsstring), &v2wms)
	assert.NoError(t, err)
	var wms pdoknlv3.WMS
	v2beta1.V3HubFromV2(&v2wms, &wms)

	pdoknlv3.SetHost("http://localhost")

	config, err := GetConfig(&wms, ownerInfo)
	assert.NoError(t, err)
	assert.Equal(t, WMSConfig, config)
}
