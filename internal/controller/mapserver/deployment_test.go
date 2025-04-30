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
	v2WfsString                = "apiVersion: pdok.nl/v2beta1\nkind: WFS\nmetadata:\n  name: rvo-nwbwegen-v1-0\n  labels:\n    dataset-owner: rvo\n    dataset: nwbwegen\n    service-version: v1_0\n    service-type: wfs\n  annotations:\n    lifecycle-phase: prod\n    service-bundle-id: 053b9c7e-85dc-535a-8a8c-f0d44c31511d\nspec:\n  general:\n    datasetOwner: rvo\n    dataset: nwbwegen\n    serviceVersion: v1_0\n  kubernetes:\n    healthCheck:\n      mimetype: text/xml\n      querystring: SERVICE=WFS&VERSION=2.0.0&REQUEST=GetCapabilities\n    resources:\n      limits:\n        ephemeralStorage: 25Mi\n      requests:\n        ephemeralStorage: 25Mi\n  service:\n    metadataIdentifier: 7c77bdc8-5011-4139-8957-d244fb9d3489\n    title: NWB - Vaarwegen WFS\n    abstract: Deze webfeatureservice bevat alleen de vaarwegvakken en kilometermarkeringen\n      van het NWB - Vaarwegen (Nationaal Wegen Bestand).\n    authority:\n      name: rws\n      url: https://www.rijkswaterstaat.nl\n    dataEPSG: EPSG:28992\n    extent: -59188.44333693248 304984.64144318487 308126.88473339565 858328.516489961\n    inspire: true\n    keywords:\n      - Vervoersnetwerken\n      - Vaarwegen\n      - Schepen\n      - Scheepvaart\n      - Vaarwegvakken\n      - Kilometermarkering\n      - HVD\n      - Mobiliteit\n    featureTypes:\n      - abstract: Deze webfeatureservice bevat alleen de kilometermarkeringen van het\n          NWB - Vaarwegen (Nationaal Wegen Bestand).\n        data:\n          gpkg:\n            blobKey: geopackages/rvo/nwbwegen/fe2b0b0e-3c88-4f74-b31c-f5915cefe530/1/nwb_vaarwegen.gpkg\n            columns:\n              - fid\n              - objectid\n              - vwk_id\n              - vwk_begdtm\n              - pos_tov_as\n              - gtlwaarde\n              - ltrwaarde\n              - afstand\n              - mst_code\n            geometryType: MultiPoint\n            table: kmmarkeringen\n        datasetMetadataIdentifier: 00d8c7c8-98ff-4b06-8f53-b44216e6e75c\n        keywords:\n          - Vervoersnetwerken\n          - Vaarwegen\n          - Schepen\n          - Scheepvaart\n          - Kilometermarkering\n        name: kmmarkeringen\n        sourceMetadataIdentifier: a757e146-09fe-4585-a236-aae0dcd6f266\n        title: Kmmarkeringen\n      - abstract: Deze webfeatureservice bevat vaarwegvakken van het NWB - Vaarwegen\n          (Nationaal Wegen Bestand).\n        data:\n          gpkg:\n            blobKey: geopackages/rvo/nwbwegen/fe2b0b0e-3c88-4f74-b31c-f5915cefe530/1/nwb_vaarwegen.gpkg\n            columns:\n              - fid\n              - objectid\n              - vwk_id\n              - vwk_begdtm\n              - vwj_id_beg\n              - vwj_id_end\n              - vaktype\n              - vrt_code\n              - vrt_naam\n              - vwg_nr\n              - vwg_naam\n              - begkm\n              - endkm\n              - st_lengthshape\n            geometryType: MultiLineString\n            table: vaarwegvakken\n        datasetMetadataIdentifier: 00d8c7c8-98ff-4b06-8f53-b44216e6e75c\n        keywords:\n          - Vervoersnetwerken\n          - Vaarwegen\n          - Schepen\n          - Scheepvaart\n          - Vaarwegvakken\n        name: vaarwegvakken\n        sourceMetadataIdentifier: a757e146-09fe-4585-a236-aae0dcd6f266\n        title: Vaarwegvakken\n"
	expectedVolumeMountsString = "[{\"name\":\"base\",\"mountPath\":\"/srv/data\"},{\"name\":\"data\",\"mountPath\":\"/var/www\"},{\"name\":\"mapserver\",\"mountPath\":\"/srv/mapserver/config/default_mapserver.conf\",\"subPath\":\"default_mapserver.conf\"},{\"name\":\"mapserver\",\"mountPath\":\"/srv/mapserver/config/include.conf\",\"subPath\":\"include.conf\"},{\"name\":\"mapserver\",\"mountPath\":\"/srv/mapserver/config/ogc.lua\",\"subPath\":\"ogc.lua\"},{\"name\":\"mapserver\",\"mountPath\":\"/srv/mapserver/config/scraping-error.xml\",\"subPath\":\"scraping-error.xml\"}]"
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
	a := 0
	_ = result
	_ = a
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
