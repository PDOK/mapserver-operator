package legendgenerator

import (
	"github.com/pdok/mapserver-operator/api/v2beta1"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestGetConfigMapDataNoLegendFix(t *testing.T) {

	v2wmsstring := "apiVersion: pdok.nl/v2beta1\nkind: WMS\nmetadata:\n  name: rws-nwbwegen-v1-0\n  labels:\n    dataset-owner: rws\n    dataset: nwbwegen\n    service-version: v1_0\n    service-type: wms\n  annotations:\n    lifecycle-phase: prod\n    service-bundle-id: b39c152b-393b-52f5-a50c-e1ffe904b6fb\nspec:\n  general:\n    datasetOwner: rws\n    dataset: nwbwegen\n    serviceVersion: v1_0\n  kubernetes:\n    healthCheck:\n      boundingbox: 135134.89,457152.55,135416.03,457187.82\n    resources:\n      limits:\n        ephemeralStorage: 1535Mi\n        memory: 4G\n      requests:\n        cpu: 2000m\n        ephemeralStorage: 1535Mi\n        memory: 4G\n  options:\n    automaticCasing: true\n    disableWebserviceProxy: false\n    includeIngress: true\n    validateRequests: true\n  service:\n    title: NWB - Wegen WMS\n    abstract:\n      Dit is de web map service van het Nationaal Wegen Bestand (NWB) - wegen.\n      Deze dataset bevat alleen de wegvakken en hectometerpunten. Het Nationaal Wegen\n      Bestand - Wegen is een digitaal geografisch bestand van alle wegen in Nederland.\n      Opgenomen zijn alle wegen die worden beheerd door wegbeheerders als het Rijk,\n      provincies, gemeenten en waterschappen, echter alleen voor zover deze zijn voorzien\n      van een straatnaam of nummer.\n    authority:\n      name: rws\n      url: https://www.rijkswaterstaat.nl\n    dataEPSG: EPSG:28992\n    extent: -59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961\n    inspire: true\n    keywords:\n      - Vervoersnetwerken\n      - Menselijke gezondheid en veiligheid\n      - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n      - Nationaal\n      - Voertuigen\n      - Verkeer\n      - Wegvakken\n      - Hectometerpunten\n      - HVD\n      - Mobiliteit\n    stylingAssets:\n      configMapRefs:\n        - name: includes\n          keys:\n            - nwb_wegen_hectopunten.symbol\n            - hectopunten.style\n            - wegvakken.style\n      blobKeys:\n        - resources/fonts/liberation-sans.ttf\n    layers:\n      - abstract:\n          Deze laag bevat de wegvakken uit het Nationaal Wegen bestand (NWB)\n          en geeft gedetailleerde informatie per wegvak zoals straatnaam, wegnummer,\n          routenummer, wegbeheerder, huisnummers, enz. weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - wvk_id\n              - wvk_begdat\n              - jte_id_beg\n              - jte_id_end\n              - wegbehsrt\n              - wegnummer\n              - wegdeelltr\n              - hecto_lttr\n              - bst_code\n              - rpe_code\n              - admrichtng\n              - rijrichtng\n              - stt_naam\n              - stt_bron\n              - wpsnaam\n              - gme_id\n              - gme_naam\n              - hnrstrlnks\n              - hnrstrrhts\n              - e_hnr_lnks\n              - e_hnr_rhts\n              - l_hnr_lnks\n              - l_hnr_rhts\n              - begafstand\n              - endafstand\n              - beginkm\n              - eindkm\n              - pos_tv_wol\n              - wegbehcode\n              - wegbehnaam\n              - distrcode\n              - distrnaam\n              - dienstcode\n              - dienstnaam\n              - wegtype\n              - wgtype_oms\n              - routeltr\n              - routenr\n              - routeltr2\n              - routenr2\n              - routeltr3\n              - routenr3\n              - routeltr4\n              - routenr4\n              - wegnr_aw\n              - wegnr_hmp\n              - geobron_id\n              - geobron_nm\n              - bronjaar\n              - openlr\n              - bag_orl\n              - frc\n              - fow\n              - alt_naam\n              - alt_nr\n              - rel_hoogte\n              - st_lengthshape\n            geometryType: MultiLineString\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: wegvakken\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Wegvakken\n        maxScale: 50000.0\n        minScale: 1.0\n        name: wegvakken\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: wegvakken\n            title: NWB - Wegvakken\n            visualization: wegvakken.style\n        title: Wegvakken\n        visible: true\n      - abstract:\n          Deze laag bevat de hectopunten uit het Nationaal Wegen Bestand (NWB)\n          en geeft gedetailleerde informatie per hectopunt zoals hectometrering, afstand,\n          zijde en hectoletter weer.\n        data:\n          gpkg:\n            columns:\n              - objectid\n              - hectomtrng\n              - afstand\n              - wvk_id\n              - wvk_begdat\n              - zijde\n              - hecto_lttr\n            geometryType: MultiPoint\n            blobKey: geopackages/rws/nwbwegen/410a6d1e-e767-41b4-ba8d-9e1e955dd013/1/nwb_wegen.gpkg\n            table: hectopunten\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Hectometerpunten\n        maxScale: 50000.0\n        minScale: 1.0\n        name: hectopunten\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        styles:\n          - name: hectopunten\n            title: NWB - Hectopunten\n            visualization: hectopunten.style\n        title: Hectopunten\n        visible: true\n    metadataIdentifier: f2437a92-ddd3-4777-a1bc-fdf4b4a7fcb8\n"
	var v2wms v2beta1.WMS
	err := yaml.Unmarshal([]byte(v2wmsstring), &v2wms)
	assert.NoError(t, err)
	var wms pdoknlv3.WMS
	v2beta1.V3WMSHubFromV2(&v2wms, &wms)

	configMapData := GetConfigMapData(&wms)

	expectedData := make(map[string]string)
	expectedData["default_mapserver.conf"] = "CONFIG\n  ENV\n    MS_MAP_NO_PATH \"true\"\n  END\nEND\n"
	expectedData["input"] = "\"wegvakken\" \"wegvakken\"\n\"hectopunten\" \"hectopunten\"\n"
	expectedData["input2"] = "- layer: wegvakken\n  style: wegvakken\n- layer: hectopunten\n  style: hectopunten\n"

	assert.Equal(t, expectedData, configMapData)
}

func TestGetConfigMapDataLegendFix(t *testing.T) {

	v2wmsstring := "apiVersion: pdok.nl/v2beta1\nkind: WMS\nmetadata:\n  name: kadaster-kadastralekaart\n  labels:\n    dataset-owner: kadaster\n    dataset: kadastralekaart\n    service-version: v5_0\n    service-type: wms\nspec:\n  general:\n    datasetOwner: kadaster\n    dataset: kadastralekaart\n    serviceVersion: v5_0\n  kubernetes:\n    healthCheck:\n      querystring: language=dut&SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&BBOX=193882.0336615453998,470528.1693874415942,193922.4213813782844,470564.250484353397&CRS=EPSG:28992&WIDTH=769&HEIGHT=687&LAYERS=OpenbareRuimteNaam,Bebouwing,Perceel,KadastraleGrens&FORMAT=image/png&DPI=96&MAP_RESOLUTION=96&FORMAT_OPTIONS=dpi:96&TRANSPARENT=TRUE\n      mimetype: image/png\n    resources:\n      limits:\n        memory: \"100M\"\n        ephemeralStorage: \"200M\"\n      requests:\n        cpu: \"500\"\n        memory: \"100M\"\n        ephemeralStorage: \"100M\"\n  options:\n    automaticCasing: true\n    disableWebserviceProxy: false\n    includeIngress: true\n    validateRequests: true\n    rewriteGroupToDataLayers: true\n  service:\n    inspire: false\n    title: Kadastrale Kaart (WMS)\n    abstract: Overzicht van de ligging van de kadastrale percelen in Nederland. Fungeert als schakel tussen terrein en registratie, vervult voor externe gebruiker vaak een referentiefunctie, een ondergrond ten opzichte waarvan de gebruiker eigen informatie kan vastleggen en presenteren.\n    keywords:\n      - Kadaster\n      - Kadastrale percelen\n      - Kadastrale grens\n      - Kadastrale kaart\n      - Bebouwing\n      - Nummeraanduidingreeks\n      - Openbare ruimte naam\n      - Perceel\n      - Grens\n      - Kwaliteit\n      - Kwaliteitslabels\n      - HVD\n      - Geospatiale data\n    metadataIdentifier: 97cf6a64-9cfc-4ce6-9741-2db44fd27fca\n    authority:\n      name: kadaster\n      url: https://www.kadaster.nl\n    dataEPSG: EPSG:28992\n    resolution: 91\n    defResolution: 91\n    extent: \"-25000 250000 280000 860000\"\n    maxSize: 10000\n    stylingAssets:\n      configMapRefs:\n        - name: ${INCLUDES}\n      blobKeys:\n        - ${BLOBS_RESOURCES_BUCKET}/fonts/liberation-sans.ttf\n        - ${BLOBS_RESOURCES_BUCKET}/fonts/liberation-sans-italic.ttf\n    layers:\n      - name: Kadastralekaart\n        title: KadastraleKaartv5\n        abstract: Overzicht van de ligging van de kadastrale percelen in Nederland. Fungeert als schakel tussen terrein en registratie, vervult voor externe gebruiker vaak een referentiefunctie, een ondergrond ten opzichte waarvan de gebruiker eigen informatie kan vastleggen en presenteren.\n        maxScale: 6001\n        keywords:\n          - Kadaster\n          - Kadastrale percelen\n          - Kadastrale grens\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n      - name: Bebouwing\n        visible: true\n        group: Kadastralekaart\n        title: Bebouwing\n        abstract: De laag Bebouwing is een selectie op panden van de BGT.\n        keywords:\n          - Bebouwing\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard:bebouwing\n            title: Standaardvisualisatie Bebouwing\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n          - name: kwaliteit:bebouwing\n            title: Kwaliteitsvisualisatie Bebouwing\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n          - name: print:bebouwing\n            title: Printvisualisatie Bebouwing\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n      - name: Bebouwingvlak\n        visible: true\n        group: Bebouwing\n        title: Bebouwingvlak\n        abstract: De laag Bebouwing is een selectie op panden van de BGT.\n        keywords:\n          - Bebouwing\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: bebouwing.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: bebouwing_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: bebouwing_print.style\n          - name: standaard:bebouwing\n            title: Standaardvisualisatie Bebouwing\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: bebouwing.group.style\n          - name: kwaliteit:bebouwing\n            title: Kwaliteitsvisualisatie Bebouwing\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: bebouwing_kwaliteit.group.style\n          - name: print:bebouwing\n            title: Printvisualisatie Bebouwing\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: bebouwing_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/pand.gpkg\n            table: pand\n            geometryType: Polygon\n            columns:\n              - object_begin_tijd\n              - lv_publicatiedatum\n              - relatieve_hoogteligging\n              - in_onderzoek\n              - tijdstip_registratie\n              - identificatie_namespace\n              - identificatie_lokaal_id\n              - bronhouder\n              - bgt_status\n              - plus_status\n              - identificatie_bag_pnd\n            aliases:\n              lv_publicatiedatum: LV-publicatiedatum\n              identificatie_lokaal_id: identificatieLokaalID\n              identificatie_bag_pnd: identificatieBAGPND\n              bgt_status: bgt-status\n              plus_status: plus-status\n      - name: Nummeraanduidingreeks\n        visible: true\n        group: Bebouwing\n        title: Nummeraanduidingreeks\n        abstract: De laag Bebouwing is een selectie op panden van de BGT.\n        keywords:\n          - Nummeraanduidingreeks\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 2001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaarvisualisatie van de nummeraanduidingreeks.\n            visualization: nummeraanduidingreeks.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: nummeraanduidingreeks_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: nummeraanduidingreeks_print.style\n          - name: standaard:bebouwing\n            title: Standaardvisualisatie Bebouwing\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: nummeraanduidingreeks.group.style\n          - name: kwaliteit:bebouwing\n            title: Kwaliteitsvisualisatie Bebouwing\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: nummeraanduidingreeks_kwaliteit.group.style\n          - name: print:bebouwing\n            title: Printvisualisatie Bebouwing\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: nummeraanduidingreeks_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/pand_nummeraanduiding.gpkg\n            table: pand_nummeraanduiding\n            geometryType: Point\n            columns:\n              - bebouwing_id\n              - hoek\n              - tekst\n              - bag_vbo_laagste_huisnummer\n              - bag_vbo_hoogste_huisnummer\n              - hoek\n            aliases:\n              bebouwing_id: bebouwingID\n              bag_vbo_laagste_huisnummer: identificatie_BAGVBOLaagsteHuisnummer\n              bag_vbo_hoogste_huisnummer: identificatie_BAGVBOHoogsteHuisnummer\n      - name: OpenbareRuimteNaam\n        visible: true\n        group: Kadastralekaart\n        title: OpenbareRuimteNaam\n        abstract: De laag Openbareruimtenaam is een selectie op de openbare ruimte labels van de BGT met een bgt-status \"bestaand\" die een classificatie (openbareruimtetype) Weg en Water hebben.\n        keywords:\n          - Openbare ruimte naam\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 2001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: openbareruimtenaam.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: openbareruimtenaam_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: openbareruimtenaam_print.style\n          - name: standaard:openbareruimtenaam\n            title: Standaardvisualisatie OpenbareRuimteNaam\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: openbareruimtenaam.group.style\n          - name: kwaliteit:openbareruimtenaam\n            title: Kwaliteitsvisualisatie OpenbareRuimteNaam\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: openbareruimtenaam_kwaliteit.group.style\n          - name: print:openbareruimtenaam\n            title: Printvisualisatie OpenbareRuimteNaam\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: openbareruimtenaam_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/openbareruimtelabel.gpkg\n            table: openbareruimtelabel\n            geometryType: Point\n            columns:\n              - object_begin_tijd\n              - lv_publicatiedatum\n              - relatieve_hoogteligging\n              - in_onderzoek\n              - tijdstip_registratie\n              - identificatie_namespace\n              - identificatie_lokaal_id\n              - bronhouder\n              - bgt_status\n              - plus_status\n              - identificatie_bag_opr\n              - tekst\n              - hoek\n              - openbare_ruimte_type\n            aliases:\n              lv_publicatiedatum: LV-publicatiedatum\n              identificatie_lokaal_id: identificatieLokaalID\n              identificatie_bag_opr: identificatieBAGOPR\n              bgt_status: bgt-status\n              plus_status: plus-status\n      - name: Perceel\n        visible: true\n        group: Kadastralekaart\n        title: Perceel\n        abstract: Een perceel is een stuk grond waarvan het Kadaster de grenzen heeft gemeten of gaat meten en dat bij het Kadaster een eigen nummer heeft. Een perceel is een begrensd deel van het Nederlands grondgebied dat kadastraal geïdentificeerd is en met kadastrale grenzen begrensd is.\n        keywords:\n          - Perceel\n          - Kadastrale percelen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard:perceel\n            title: Standaardvisualisatie Perceel\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n          - name: kwaliteit:perceel\n            title: Kwaliteitsvisualisatie Perceel\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n          - name: print:perceel\n            title: Printvisualisatie Perceel\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n      - name: Perceelvlak\n        visible: true\n        group: Perceel\n        title: Perceelvlak\n        abstract: Een perceel is een stuk grond waarvan het Kadaster de grenzen heeft gemeten of gaat meten en dat bij het Kadaster een eigen nummer heeft. Een perceel is een begrensd deel van het Nederlands grondgebied dat kadastraal geïdentificeerd is en met kadastrale grenzen begrensd is.\n        keywords:\n          - Kadastrale percelen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: perceelvlak.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: perceelvlak_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: perceelvlak_print.style\n          - name: standaard:perceel\n            title: Standaardvisualisatie Perceel\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: perceelvlak.group.style\n          - name: kwaliteit:perceel\n            title: Kwaliteitsvisualisatie Perceel\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: perceelvlak_kwaliteit.group.style\n          - name: print:perceel\n            title: Printvisualisatie Perceel\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: perceelvlak_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/perceel.gpkg\n            table: perceel\n            geometryType: Polygon\n            columns:\n              - identificatie_namespace\n              - identificatie_lokaal_id\n              - begin_geldigheid\n              - tijdstip_registratie\n              - volgnummer\n              - status_historie_code\n              - status_historie_waarde\n              - kadastrale_gemeente_code\n              - kadastrale_gemeente_waarde\n              - sectie\n              - akr_kadastrale_gemeente_code_code\n              - akr_kadastrale_gemeente_code_waarde\n              - kadastrale_grootte_waarde\n              - soort_grootte_code\n              - soort_grootte_waarde\n              - perceelnummer\n              - perceelnummer_rotatie\n              - perceelnummer_verschuiving_delta_x\n              - perceelnummer_verschuiving_delta_y\n              - perceelnummer_plaatscoordinaat_x\n              - perceelnummer_plaatscoordinaat_y\n            aliases:\n              identificatie_lokaal_id: identificatieLokaalID\n              akr_kadastrale_gemeente_code_code: AKRKadastraleGemeenteCodeCode\n              akr_kadastrale_gemeente_code_waarde: AKRKadastraleGemeenteCodeWaarde\n      - name: Label\n        visible: true\n        group: Perceel\n        title: Label\n        abstract: Een perceel is een stuk grond waarvan het Kadaster de grenzen heeft gemeten of gaat meten en dat bij het Kadaster een eigen nummer heeft. Een perceel is een begrensd deel van het Nederlands grondgebied dat kadastraal geïdentificeerd is en met kadastrale grenzen begrensd is.\n        keywords:\n          - Kadastrale percelen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaarvisualisatie van het label.\n            visualization: label.style\n          - name: standaard:perceel\n            title: Standaardvisualisatie Perceel\n            abstract: Standaarvisualisatie van het label.\n            visualization: label.group.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: label_kwaliteit.style\n          - name: kwaliteit:perceel\n            title: Kwaliteitsvisualisatie Perceel\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: label_kwaliteit.group.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: label_print.style\n          - name: print:perceel\n            title: Printvisualisatie Perceel\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: label_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/perceel_label.gpkg\n            table: perceel_label\n            geometryType: Point\n            columns:\n              - perceel_id\n              - perceelnummer\n              - rotatie\n              - verschuiving_delta_x\n              - verschuiving_delta_y\n            aliases:\n              perceel_id: perceelID\n      - name: Bijpijling\n        visible: true\n        group: Perceel\n        title: Bijpijling\n        abstract: Een perceel is een stuk grond waarvan het Kadaster de grenzen heeft gemeten of gaat meten en dat bij het Kadaster een eigen nummer heeft. Een perceel is een begrensd deel van het Nederlands grondgebied dat kadastraal geïdentificeerd is en met kadastrale grenzen begrensd is.\n        keywords:\n          - Kadastrale percelen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: bijpijling.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: bijpijling_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: bijpijling_print.style\n          - name: standaard:perceel\n            title: Standaardvisualisatie Perceel\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: bijpijling.group.style\n          - name: kwaliteit:perceel\n            title: Kwaliteitsvisualisatie Perceel\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: bijpijling_kwaliteit.group.style\n          - name: print:perceel\n            title: Printvisualisatie Perceel\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: bijpijling_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/perceel_bijpijling.gpkg\n            table: perceel_bijpijling\n            geometryType: LineString\n            columns:\n              - perceel_id\n            aliases:\n              perceel_id: perceelID\n      - name: KadastraleGrens\n        visible: true\n        group: Kadastralekaart\n        title: KadastraleGrens\n        abstract: Een Kadastrale Grens is de weergave van een grens op de kadastrale kaart die door de dienst van het Kadaster tussen percelen (voorlopig) vastgesteld wordt, op basis van inlichtingen van belanghebbenden en met  gebruikmaking van de aan de kadastrale kaart ten grondslag liggende bescheiden die in elk geval de landmeetkundige gegevens bevatten van hetgeen op die kaart wordt weergegeven.\n        keywords:\n          - Grens\n          - Kadastrale grenzen\n        datasetMetadataIdentifier: a29917b9-3426-4041-a11b-69bcb2256904\n        sourceMetadataIdentifier: 06b6c650-cdb1-11dd-ad8b-0800200c9a64\n        minScale: 50\n        maxScale: 6001\n        styles:\n          - name: standaard\n            title: Standaardvisualisatie\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: kadastralegrens.style\n          - name: kwaliteit\n            title: Kwaliteitsvisualisatie\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: kadastralegrens_kwaliteit.style\n          - name: print\n            title: Printvisualisatie\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: kadastralegrens_print.style\n          - name: standaard:kadastralegrens\n            title: Standaardvisualisatie KadastraleGrens\n            abstract: Standaardvisualisatie met grenzen op basis van type (definitief, voorlopig of administratief).\n            visualization: kadastralegrens.group.style\n          - name: kwaliteit:kadastralegrens\n            title: Kwaliteitsvisualisatie KadastraleGrens\n            abstract: Kwaliteitsvisualisatie met grenzen op basis van kwaliteitsklasse (B, C, D of E).\n            visualization: kadastralegrens_kwaliteit.group.style\n          - name: print:kadastralegrens\n            title: Printvisualisatie KadastraleGrens\n            abstract: Visualisatie ten behoeve van afdrukken op 180 dpi.\n            visualization: kadastralegrens_print.group.style\n        data:\n          gpkg:\n            blobKey: ${BLOBS_GEOPACKAGES_BUCKET}/kadaster/kadastralekaart_brk/${GPKG_VERSION}/kadastrale_grens.gpkg\n            table: kadastrale_grens\n            geometryType: LineString\n            columns:\n              - begin_geldigheid\n              - tijdstip_registratie\n              - volgnummer\n              - status_historie_code\n              - status_historie_waarde\n              - identificatie_namespace\n              - identificatie_lokaal_id\n              - type_grens_code\n              - type_grens_waarde\n              - classificatie_kwaliteit_code\n              - classificatie_kwaliteit_waarde\n              - perceel_links_identificatie_namespace\n              - perceel_links_identificatie_lokaal_id\n              - perceel_rechts_identificatie_namespace\n              - perceel_rechts_identificatie_lokaal_id\n            aliases:\n              identificatie_lokaal_id: identificatieLokaalID\n              perceel_links_identificatie_lokaal_id: perceelLinksIdentificatieLokaalID\n              perceel_rechts_identificatie_lokaal_id: perceelRechtsIdentificatieLokaalID\n              classificatie_kwaliteit_code: ClassificatieKwaliteitCode\n              classificatie_kwaliteit_waarde: ClassificatieKwaliteitWaarde\n"
	var v2wms v2beta1.WMS
	err := yaml.Unmarshal([]byte(v2wmsstring), &v2wms)
	assert.NoError(t, err)
	var wms pdoknlv3.WMS
	v2beta1.V3WMSHubFromV2(&v2wms, &wms)

	configMapData := GetConfigMapData(&wms)

	expectedData := make(map[string]string)
	expectedData["default_mapserver.conf"] = "CONFIG\n  ENV\n    MS_MAP_NO_PATH \"true\"\n  END\nEND\n"
	expectedData["input"] = "\"Bebouwing\" \"standaard:bebouwing\"\n\"Bebouwing\" \"kwaliteit:bebouwing\"\n\"Bebouwing\" \"print:bebouwing\"\n\"Bebouwingvlak\" \"standaard\"\n\"Bebouwingvlak\" \"kwaliteit\"\n\"Bebouwingvlak\" \"print\"\n\"Bebouwingvlak\" \"standaard:bebouwing\"\n\"Bebouwingvlak\" \"kwaliteit:bebouwing\"\n\"Bebouwingvlak\" \"print:bebouwing\"\n\"Nummeraanduidingreeks\" \"standaard\"\n\"Nummeraanduidingreeks\" \"kwaliteit\"\n\"Nummeraanduidingreeks\" \"print\"\n\"Nummeraanduidingreeks\" \"standaard:bebouwing\"\n\"Nummeraanduidingreeks\" \"kwaliteit:bebouwing\"\n\"Nummeraanduidingreeks\" \"print:bebouwing\"\n\"OpenbareRuimteNaam\" \"standaard\"\n\"OpenbareRuimteNaam\" \"kwaliteit\"\n\"OpenbareRuimteNaam\" \"print\"\n\"OpenbareRuimteNaam\" \"standaard:openbareruimtenaam\"\n\"OpenbareRuimteNaam\" \"kwaliteit:openbareruimtenaam\"\n\"OpenbareRuimteNaam\" \"print:openbareruimtenaam\"\n\"Perceel\" \"standaard:perceel\"\n\"Perceel\" \"kwaliteit:perceel\"\n\"Perceel\" \"print:perceel\"\n\"Perceelvlak\" \"standaard\"\n\"Perceelvlak\" \"kwaliteit\"\n\"Perceelvlak\" \"print\"\n\"Perceelvlak\" \"standaard:perceel\"\n\"Perceelvlak\" \"kwaliteit:perceel\"\n\"Perceelvlak\" \"print:perceel\"\n\"Label\" \"standaard\"\n\"Label\" \"standaard:perceel\"\n\"Label\" \"kwaliteit\"\n\"Label\" \"kwaliteit:perceel\"\n\"Label\" \"print\"\n\"Label\" \"print:perceel\"\n\"Bijpijling\" \"standaard\"\n\"Bijpijling\" \"kwaliteit\"\n\"Bijpijling\" \"print\"\n\"Bijpijling\" \"standaard:perceel\"\n\"Bijpijling\" \"kwaliteit:perceel\"\n\"Bijpijling\" \"print:perceel\"\n\"KadastraleGrens\" \"standaard\"\n\"KadastraleGrens\" \"kwaliteit\"\n\"KadastraleGrens\" \"print\"\n\"KadastraleGrens\" \"standaard:kadastralegrens\"\n\"KadastraleGrens\" \"kwaliteit:kadastralegrens\"\n\"KadastraleGrens\" \"print:kadastralegrens\"\n"
	expectedData["input2"] = "- layer: Bebouwing\n  style: standaard:bebouwing\n- layer: Bebouwing\n  style: kwaliteit:bebouwing\n- layer: Bebouwing\n  style: print:bebouwing\n- layer: Bebouwingvlak\n  style: standaard\n- layer: Bebouwingvlak\n  style: kwaliteit\n- layer: Bebouwingvlak\n  style: print\n- layer: Bebouwingvlak\n  style: standaard:bebouwing\n- layer: Bebouwingvlak\n  style: kwaliteit:bebouwing\n- layer: Bebouwingvlak\n  style: print:bebouwing\n- layer: Nummeraanduidingreeks\n  style: standaard\n- layer: Nummeraanduidingreeks\n  style: kwaliteit\n- layer: Nummeraanduidingreeks\n  style: print\n- layer: Nummeraanduidingreeks\n  style: standaard:bebouwing\n- layer: Nummeraanduidingreeks\n  style: kwaliteit:bebouwing\n- layer: Nummeraanduidingreeks\n  style: print:bebouwing\n- layer: OpenbareRuimteNaam\n  style: standaard\n- layer: OpenbareRuimteNaam\n  style: kwaliteit\n- layer: OpenbareRuimteNaam\n  style: print\n- layer: OpenbareRuimteNaam\n  style: standaard:openbareruimtenaam\n- layer: OpenbareRuimteNaam\n  style: kwaliteit:openbareruimtenaam\n- layer: OpenbareRuimteNaam\n  style: print:openbareruimtenaam\n- layer: Perceel\n  style: standaard:perceel\n- layer: Perceel\n  style: kwaliteit:perceel\n- layer: Perceel\n  style: print:perceel\n- layer: Perceelvlak\n  style: standaard\n- layer: Perceelvlak\n  style: kwaliteit\n- layer: Perceelvlak\n  style: print\n- layer: Perceelvlak\n  style: standaard:perceel\n- layer: Perceelvlak\n  style: kwaliteit:perceel\n- layer: Perceelvlak\n  style: print:perceel\n- layer: Label\n  style: standaard\n- layer: Label\n  style: standaard:perceel\n- layer: Label\n  style: kwaliteit\n- layer: Label\n  style: kwaliteit:perceel\n- layer: Label\n  style: print\n- layer: Label\n  style: print:perceel\n- layer: Bijpijling\n  style: standaard\n- layer: Bijpijling\n  style: kwaliteit\n- layer: Bijpijling\n  style: print\n- layer: Bijpijling\n  style: standaard:perceel\n- layer: Bijpijling\n  style: kwaliteit:perceel\n- layer: Bijpijling\n  style: print:perceel\n- layer: KadastraleGrens\n  style: standaard\n- layer: KadastraleGrens\n  style: kwaliteit\n- layer: KadastraleGrens\n  style: print\n- layer: KadastraleGrens\n  style: standaard:kadastralegrens\n- layer: KadastraleGrens\n  style: kwaliteit:kadastralegrens\n- layer: KadastraleGrens\n  style: print:kadastralegrens\n"
	expectedData["legend-fixer.sh"] = "#!/usr/bin/env bash\nset -eo pipefail\necho \"creating legends for root and group layers by concatenating data layers\"\ninput_filepath=\"/input/input\"\nremove_filepath=\"/input/remove\"\nconfig_filepath=\"/input/ogc-webservice-proxy-config.yaml\"\nlegend_dir=\"/var/www/legend\"\n< \"${input_filepath}\" xargs -n 2 echo | while read -r layer style; do\n  export layer\n  # shellcheck disable=SC2016 # dollar is for yq\n  if ! < \"${config_filepath}\" yq -e 'env(layer) as $layer | .grouplayers | keys | contains([$layer])' &>/dev/null; then\n    continue\n  fi\n  export grouplayer=\"${layer}\"\n  grouplayer_style_filepath=\"${legend_dir}/${grouplayer}/${style}.png\"\n  # shellcheck disable=SC2016 # dollar is for yq\n  datalayers=$(< \"${config_filepath}\" yq 'env(grouplayer) as $foo | .grouplayers[$foo][]')\n  datalayer_style_filepaths=()\n  for datalayer in $datalayers; do\n    datalayer_style_filepath=\"${legend_dir}/${datalayer}/${style}.png\"\n    if [[ -f \"${datalayer_style_filepath}\" ]]; then\n      datalayer_style_filepaths+=(\"${datalayer_style_filepath}\")\n    fi\n  done\n  if [[ -n \"${datalayer_style_filepaths[*]}\" ]]; then\n    echo \"concatenating ${grouplayer_style_filepath}\"\n    gm convert -append \"${datalayer_style_filepaths[@]}\" \"${grouplayer_style_filepath}\"\n  else\n    echo \"no data for ${grouplayer_style_filepath}\"\n  fi\ndone\n< \"${remove_filepath}\" xargs -n 2 echo | while read -r layer style; do\n  remove_legend_file=\"${legend_dir}/${layer}/${style}.png\"\n  echo removing $remove_legend_file\n  rm $remove_legend_file\ndone\necho \"done\""
	expectedData["ogc-webservice-proxy-config.yaml"] = "grouplayers:\n  Bebouwing:\n  - Bebouwingvlak\n  - Nummeraanduidingreeks\n  Kadastralekaart:\n  - Bebouwingvlak\n  - Nummeraanduidingreeks\n  - OpenbareRuimteNaam\n  - Perceelvlak\n  - Label\n  - Bijpijling\n  - KadastraleGrens\n  Perceel:\n  - Perceelvlak\n  - Label\n  - Bijpijling\n"
	expectedData["remove"] = "\"OpenbareRuimteNaam\" \"standaard\"\n\"OpenbareRuimteNaam\" \"kwaliteit\"\n\"OpenbareRuimteNaam\" \"print\"\n\"KadastraleGrens\" \"standaard\"\n\"KadastraleGrens\" \"kwaliteit\"\n\"KadastraleGrens\" \"print\"\n"

	assert.Equal(t, expectedData, configMapData)
}
