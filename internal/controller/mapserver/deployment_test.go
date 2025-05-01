package mapserver

import (
	"encoding/json"
	"github.com/pdok/mapserver-operator/api/v2beta1"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
	"testing"
)

const (
	v2WfsString                = "apiVersion: pdok.nl/v2beta1\nkind: WFS\nmetadata:\n  name: rws-nwbwegen-v1-0\n  labels:\n    dataset-owner: rws\n    dataset: nwbwegen\n    service-version: v1_0\n    service-type: wfs\n  annotations:\n    lifecycle-phase: prod\n    service-bundle-id: b39c152b-393b-52f5-a50c-e1ffe904b6fb\nspec:\n  general:\n    datasetOwner: rws\n    dataset: nwbwegen\n    serviceVersion: v1_0\n  kubernetes:\n    healthCheck:\n      mimetype: text/xml\n      querystring: SERVICE=WFS&VERSION=2.0.0&REQUEST=GetCapabilities\n    resources:\n      limits:\n        ephemeralStorage: 1505Mi\n      requests:\n        ephemeralStorage: 1505Mi\n  service:\n    title: NWB - Wegen WFS\n    abstract:\n      Dit is de web feature service van het Nationaal Wegen Bestand (NWB)\n      - wegen. Deze dataset bevat alleen de wegvakken en hectometerpunten. Het Nationaal\n      Wegen Bestand - Wegen is een digitaal geografisch bestand van alle wegen in\n      Nederland. Opgenomen zijn alle wegen die worden beheerd door wegbeheerders als\n      het Rijk, provincies, gemeenten en waterschappen, echter alleen voor zover deze\n      zijn voorzien van een straatnaam of nummer.\n    inspire: true\n    metadataIdentifier: a9fa7fff-6365-4885-950c-e9d9848359ee\n    authority:\n      name: rws\n      url: https://www.rijkswaterstaat.nl\n    dataEPSG: EPSG:28992\n    extent: -59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961\n    keywords:\n      - Vervoersnetwerken\n      - Menselijke gezondheid en veiligheid\n      - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n      - Nationaal\n      - Voertuigen\n      - Verkeer\n      - Wegvakken\n      - Hectometerpunten\n      - HVD\n      - Mobiliteit\n    featureTypes:\n      - name: wegvakken\n        title: Wegvakken\n        abstract:\n          Dit featuretype bevat de wegvakken uit het Nationaal Wegen bestand\n          (NWB) en bevat gedetailleerde informatie per wegvak zoals straatnaam, wegnummer,\n          routenummer, wegbeheerder, huisnummers, enz.\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Wegvakken\n        data:\n          gpkg:\n            table: wegvakken\n            geometryType: MultiLineString\n            blobKey: geopackages/rws/nwbwegen/1c56dc48-2cf4-4631-8b09-ed385d5368d1/1/nwb_wegen.gpkg\n            columns:\n              - fid\n              - objectid\n              - wvk_id\n              - wvk_begdat\n              - jte_id_beg\n              - jte_id_end\n              - wegbehsrt\n              - wegnummer\n              - wegdeelltr\n              - hecto_lttr\n              - bst_code\n              - rpe_code\n              - admrichtng\n              - rijrichtng\n              - stt_naam\n              - stt_bron\n              - wpsnaam\n              - gme_id\n              - gme_naam\n              - hnrstrlnks\n              - hnrstrrhts\n              - e_hnr_lnks\n              - e_hnr_rhts\n              - l_hnr_lnks\n              - l_hnr_rhts\n              - begafstand\n              - endafstand\n              - beginkm\n              - eindkm\n              - pos_tv_wol\n              - wegbehcode\n              - wegbehnaam\n              - distrcode\n              - distrnaam\n              - dienstcode\n              - dienstnaam\n              - wegtype\n              - wgtype_oms\n              - routeltr\n              - routenr\n              - routeltr2\n              - routenr2\n              - routeltr3\n              - routenr3\n              - routeltr4\n              - routenr4\n              - wegnr_aw\n              - wegnr_hmp\n              - geobron_id\n              - geobron_nm\n              - bronjaar\n              - openlr\n              - bag_orl\n              - frc\n              - fow\n              - alt_naam\n              - alt_nr\n              - rel_hoogte\n              - st_lengthshape\n      - name: hectopunten\n        title: Hectopunten\n        abstract:\n          Dit featuretype bevat de hectopunten uit het Nationaal Wegen Bestand\n          (NWB) en bevat gedetailleerde informatie per hectopunt zoals hectometrering,\n          afstand, zijde en hectoletter.\n        sourceMetadataIdentifier: 8f0497f0-dbd7-4bee-b85a-5fdec484a7ff\n        datasetMetadataIdentifier: a9b7026e-0a81-4813-93bd-ba49e6f28502\n        keywords:\n          - Vervoersnetwerken\n          - Menselijke gezondheid en veiligheid\n          - Geluidsbelasting hoofdwegen (Richtlijn Omgevingslawaai)\n          - Nationaal\n          - Voertuigen\n          - Verkeer\n          - Hectometerpunten\n        data:\n          gpkg:\n            blobKey: geopackages/rws/nwbwegen/1c56dc48-2cf4-4631-8b09-ed385d5368d1/1/nwb_wegen.gpkg\n            columns:\n              - fid\n              - objectid\n              - hectomtrng\n              - afstand\n              - wvk_id\n              - wvk_begdat\n              - zijde\n              - hecto_lttr\n            geometryType: MultiPoint\n            table: hectopunten\n"
	expectedVolumeMountsString = "[{\"name\":\"base\",\"mountPath\":\"/srv/data\"},{\"name\":\"data\",\"mountPath\":\"/var/www\"},{\"name\":\"mapserver\",\"mountPath\":\"/srv/mapserver/config/default_mapserver.conf\",\"subPath\":\"default_mapserver.conf\"},{\"name\":\"mapserver\",\"mountPath\":\"/srv/mapserver/config/include.conf\",\"subPath\":\"include.conf\"},{\"name\":\"mapserver\",\"mountPath\":\"/srv/mapserver/config/ogc.lua\",\"subPath\":\"ogc.lua\"},{\"name\":\"mapserver\",\"mountPath\":\"/srv/mapserver/config/scraping-error.xml\",\"subPath\":\"scraping-error.xml\"}]"
	expectedEnvVarsString      = "[{\"name\":\"SERVICE_TYPE\",\"value\":\"WFS\"},{\"name\":\"MAPSERVER_CONFIG_FILE\",\"value\":\"/srv/mapserver/config/default_mapserver.conf\"},{\"name\":\"AZURE_STORAGE_CONNECTION_STRING\",\"valueFrom\":{\"secretKeyRef\":{\"name\":\"blobs-secret\",\"key\":\"AZURE_STORAGE_CONNECTION_STRING\"}}},{\"name\":\"MS_MAPFILE\",\"value\":\"/srv/data/config/mapfile/service.map\"}]\n"
)

func TestGetVolumeMountsForDeployment(t *testing.T) {
	var wfs = getV3()
	pdoknlv3.SetHost("https://service.pdok.nl")
	result := GetVolumeMountsForDeployment(wfs, "/srv")

	var expectedVolumeMounts []v1.VolumeMount
	err := json.Unmarshal([]byte(expectedVolumeMountsString), &expectedVolumeMounts)
	assert.NoError(t, err)
	assert.Equal(t, expectedVolumeMounts, result)
}

func TestGetEnvVarsForDeployment(t *testing.T) {
	var wfs = getV3()
	pdoknlv3.SetHost("https://service.pdok.nl")
	result := GetEnvVarsForDeployment(wfs, "blobs-secret")
	var expectedEnvVars []v1.EnvVar
	err := json.Unmarshal([]byte(expectedEnvVarsString), &expectedEnvVars)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnvVars, result)
}

func TestGetResourcesForDeployment(t *testing.T) {
	var wfs = getV3()
	pdoknlv3.SetHost("https://service.pdok.nl")
	result := GetResourcesForDeployment(wfs)
	marshalled, _ := json.Marshal(result)
	println(string(marshalled))
}

func getV3() *pdoknlv3.WFS {
	var v2wfs v2beta1.WFS
	err := yaml.Unmarshal([]byte(v2WfsString), &v2wfs)
	if err != nil {
		panic(err)
	}
	var wfs pdoknlv3.WFS
	v2beta1.V3WFSHubFromV2(&v2wfs, &wfs)
	return &wfs
}
