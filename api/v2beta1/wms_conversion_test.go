package v2beta1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestV2ToV3(t *testing.T) {
	//nolint:misspell
	input := "apiVersion: pdok.nl/v2beta1\nkind: WMS\nmetadata:\n  name: rws-nwbwegen-v1-0\n  labels:\n    dataset-owner: rws\n    dataset: nwbwegen\n    service-version: v1_0\n    service-type: wms\n  annotations:\n    lifecycle-phase: prod\n    service-bundle-id: b39c152b-393b-52f5-a50c-e1ffe904b6fb\nspec:\n  general:\n    datasetOwner: rws\n    dataset: nwbwegen\n    serviceVersion: v1_0\n  kubernetes:\n    healthCheck:\n      boundingbox: 135134.89,457152.55,135416.03,457187.82\n    resources:\n      limits:\n        ephemeralStorage: 1535Mi\n        memory: 4G\n      requests:\n        cpu: 2000m\n        ephemeralStorage: 1535Mi\n        memory: 4G\n  options:\n    automaticCasing: true\n    disableWebserviceProxy: false\n    includeIngress: true\n    validateRequests: true\n  service:\n    title: NWB - Wegen WMS\n    abstract:\n      Dit is de web map service van het Nationaal Wegen Bestand (NWB) - wegen.\n      Deze dataset bevat alleen de wegvakken en hectometerpunten. Het Nationaal Wegen\n      Bestand - Wegen is een digitaal geografisch bestand van alle wegen in Nederland.\n      Opgenomen zijn alle wegen die worden beheerd door wegbeheerders als het Rijk,\n      provincies, gemeenten en waterschappen, echter alleen voor zover deze zijn voorzien\n      van een straatnaam of nummer.\n    authority:\n      name: rws\n      url: https://www.rijkswaterstaat.nl\n    dataEPSG: EPSG:28992\n    extent: -59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961\n    inspire: true\n    keywords:\n      - Vervoersnetwerken\n      - Menselijke gezondheid en veiligheid\n      - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n      - Nationaal\n      - Voertuigen\n      - Verkeer\n      - Wegvakken\n      - Hectometerpunten\n      - HVD\n      - Mobiliteit\n    stylingAssets:\n      configMapRefs:\n        - name: includes\n          keys:\n            - nwb_wegen_hectopunten.symbol\n            - hectopunten.style\n            - wegvakken.style\n      blobKeys:\n        - resources/fonts/liberation-sans.ttf\n    layers:\n      - abstract:\n          Deze laag bevat de wegvakken uit het Nationaal Wegen bestand (NWB)\n          en geeft gedetailleerde informatie per wegvak zoals straatnaam, wegnummer,\n          routenummer, wegbeheerder, huisnummers, enz. weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - wvk_id\n              - wvk_begdat\n              - jte_id_beg\n              - jte_id_end\n              - wegbehsrt\n              - wegnummer\n              - wegdeelltr\n              - hecto_lttr\n              - bst_code\n              - rpe_code\n              - admrichtng\n              - rijrichtng\n              - stt_naam\n              - stt_bron\n              - wpsnaam\n              - gme_id\n              - gme_naam\n              - hnrstrlnks\n              - hnrstrrhts\n              - e_hnr_lnks\n              - e_hnr_rhts\n              - l_hnr_lnks\n              - l_hnr_rhts\n              - begafstand\n              - endafstand\n              - beginkm\n              - eindkm\n              - pos_tv_wol\n              - wegbehcode\n              - wegbehnaam\n              - distrcode\n              - distrnaam\n              - dienstcode\n              - dienstnaam\n              - wegtype\n              - wgtype_oms\n              - routeltr\n              - routenr\n              - routeltr2\n              - routenr2\n              - routeltr3\n              - routenr3\n              - routeltr4\n              - routenr4\n              - wegnr_aw\n              - wegnr_hmp\n              - geobron_id\n              - geobron_nm\n              - bronjaar\n              - openlr\n              - bag_orl\n              - frc\n              - fow\n              - alt_naam\n              - alt_nr\n              - rel_hoogte\n              - st_lengthshape\n            geometryType: MultiLineString\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: wegvakken\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Wegvakken\n        maxScale: 50000.0\n        minScale: 1.0\n        name: wegvakken\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: wegvakken\n            title: NWB - Wegvakken\n            visualization: wegvakken.style\n        title: Wegvakken\n        visible: true\n      - abstract:\n          Deze laag bevat de hectopunten uit het Nationaal Wegen Bestand (NWB)\n          en geeft gedetailleerde informatie per hectopunt zoals hectometrering, afstand,\n          zijde en hectoletter weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - hectomtrng\n              - afstand\n              - wvk_id\n              - wvk_begdat\n              - zijde\n              - hecto_lttr\n            geometryType: MultiPoint\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: hectopunten\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Hectometerpunten\n        maxScale: 50000.0\n        minScale: 1.0\n        name: hectopunten\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: hectopunten\n            title: NWB - Hectopunten\n            visualization: hectopunten.style\n        title: Hectopunten\n        visible: true\n    metadataIdentifier: f2437a92-ddd3-4777-a1bc-fdf4b4a7fcb8\n"
	v2wms := &WMS{}
	err := yaml.Unmarshal([]byte(input), v2wms)
	assert.NoError(t, err)
	var target pdoknlv3.WMS
	err = v2wms.ToV3(&target)
	assert.NoError(t, err)
	assert.Equal(t, "NWB - Wegen WMS", target.Spec.Service.Title)
	a := 0
	_ = a
}

func TestWMSService_MapLayersToV3(t *testing.T) {
	tests := []struct {
		name      string
		v2Service WMSService
		want      pdoknlv3.Layer
	}{
		{
			name: "no toplayer, middle: 1 data layer",
			v2Service: WMSService{Layers: []WMSLayer{
				{Name: "layer"},
			}},
			want: pdoknlv3.Layer{
				Title:         ptr.To(""),
				Abstract:      ptr.To(""),
				BoundingBoxes: getDefaultWMSLayerBoundingBoxes(nil),
				Visible:       true,
				Layers: []pdoknlv3.Layer{{
					Name:          ptr.To("layer"),
					BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
					Visible:       true,
					Styles:        []pdoknlv3.Style{},
				}},
			},
		},
		{
			name: "no toplayer, middle: 1 group layer",
			v2Service: WMSService{Layers: []WMSLayer{
				{Name: "group-layer"},
				{Name: "sub-layer", Group: ptr.To("group-layer")},
			}},
			want: pdoknlv3.Layer{
				Title:         ptr.To(""),
				Abstract:      ptr.To(""),
				BoundingBoxes: getDefaultWMSLayerBoundingBoxes(nil),
				Visible:       true,
				Layers: []pdoknlv3.Layer{{
					Name:          ptr.To("group-layer"),
					BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
					Visible:       true,
					Styles:        []pdoknlv3.Style{},
					Layers: []pdoknlv3.Layer{
						{
							Name:          ptr.To("sub-layer"),
							BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
							Visible:       true,
							Styles:        []pdoknlv3.Style{},
						},
					},
				}},
			},
		},
		{
			name: "no toplayer, middle: 2 group layers",
			v2Service: WMSService{Layers: []WMSLayer{
				{Name: "group-layer-1"},
				{Name: "sub-layer-1", Group: ptr.To("group-layer-1")},
				{Name: "group-layer-2"},
				{Name: "sub-layer-2", Group: ptr.To("group-layer-2")},
			}},
			want: pdoknlv3.Layer{
				Title:         ptr.To(""),
				Abstract:      ptr.To(""),
				BoundingBoxes: getDefaultWMSLayerBoundingBoxes(nil),
				Visible:       true,
				Layers: []pdoknlv3.Layer{
					{
						Name:          ptr.To("group-layer-1"),
						BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
						Visible:       true,
						Styles:        []pdoknlv3.Style{},
						Layers: []pdoknlv3.Layer{
							{
								Name:          ptr.To("sub-layer-1"),
								BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
								Visible:       true,
								Styles:        []pdoknlv3.Style{},
							},
						},
					},
					{
						Name:          ptr.To("group-layer-2"),
						BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
						Visible:       true,
						Styles:        []pdoknlv3.Style{},
						Layers: []pdoknlv3.Layer{
							{
								Name:          ptr.To("sub-layer-2"),
								BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
								Visible:       true,
								Styles:        []pdoknlv3.Style{},
							},
						},
					},
				},
			},
		},
		{
			name: "no toplayer, middle: 1 group layer, 1 data layer",
			v2Service: WMSService{Layers: []WMSLayer{
				{Name: "group-layer"},
				{Name: "sub-layer", Group: ptr.To("group-layer")},
				{Name: "data-layer"},
			}},
			want: pdoknlv3.Layer{
				Title:         ptr.To(""),
				Abstract:      ptr.To(""),
				BoundingBoxes: getDefaultWMSLayerBoundingBoxes(nil),
				Visible:       true,
				Layers: []pdoknlv3.Layer{
					{
						Name:          ptr.To("group-layer"),
						BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
						Visible:       true,
						Styles:        []pdoknlv3.Style{},
						Layers: []pdoknlv3.Layer{
							{
								Name:          ptr.To("sub-layer"),
								BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
								Visible:       true,
								Styles:        []pdoknlv3.Style{},
							},
						},
					},
					{
						Name:          ptr.To("data-layer"),
						BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
						Visible:       true,
						Styles:        []pdoknlv3.Style{},
					},
				},
			},
		},
		{
			name: "toplayer, middle: 1 group layer",
			v2Service: WMSService{Layers: []WMSLayer{
				{Name: "group-layer", Group: ptr.To("top-layer")},
				{Name: "sub-layer", Group: ptr.To("group-layer")},
				{Name: "top-layer"},
			}},
			want: pdoknlv3.Layer{
				Name:          ptr.To("top-layer"),
				BoundingBoxes: getDefaultWMSLayerBoundingBoxes(nil),
				Visible:       true,
				Styles:        []pdoknlv3.Style{},
				Layers: []pdoknlv3.Layer{{
					Name:          ptr.To("group-layer"),
					BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
					Visible:       true,
					Styles:        []pdoknlv3.Style{},
					Layers: []pdoknlv3.Layer{
						{
							Name:          ptr.To("sub-layer"),
							BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
							Visible:       true,
							Styles:        []pdoknlv3.Style{},
						},
					},
				}},
			},
		},
		{
			name: "toplayer, middle: 1 group layer, 1 data layer",
			v2Service: WMSService{Layers: []WMSLayer{
				{Name: "group-layer", Group: ptr.To("top-layer")},
				{Name: "sub-layer", Group: ptr.To("group-layer")},
				{Name: "top-layer"},
				{Name: "data-layer", Group: ptr.To("top-layer")},
			}},
			want: pdoknlv3.Layer{
				Name:          ptr.To("top-layer"),
				BoundingBoxes: getDefaultWMSLayerBoundingBoxes(nil),
				Visible:       true,
				Styles:        []pdoknlv3.Style{},
				Layers: []pdoknlv3.Layer{
					{
						Name:          ptr.To("group-layer"),
						BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
						Visible:       true,
						Styles:        []pdoknlv3.Style{},
						Layers: []pdoknlv3.Layer{
							{
								Name:          ptr.To("sub-layer"),
								BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
								Visible:       true,
								Styles:        []pdoknlv3.Style{},
							},
						},
					},
					{
						Name:          ptr.To("data-layer"),
						BoundingBoxes: []pdoknlv3.WMSBoundingBox{},
						Visible:       true,
						Styles:        []pdoknlv3.Style{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := cmp.Diff(tt.want, tt.v2Service.MapLayersToV3())
			assert.Equal(t, diff == "", true, "%s", diff)
		})
	}
}
