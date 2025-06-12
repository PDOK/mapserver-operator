package mapfilegenerator

import (
	"encoding/json"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/google/go-cmp/cmp"

	"github.com/pdok/mapserver-operator/api/v2beta1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

func TestGetConfigForWFS(t *testing.T) {
	pdoknlv3.SetHost("https://service.pdok.nl")
	ownerInfo := &smoothoperatorv1.OwnerInfo{
		Spec: smoothoperatorv1.OwnerInfoSpec{
			NamespaceTemplate: smoothoperatorutils.Pointer("http://{{prefix}}.geonovum.nl"),
		},
	}

	input, err := os.ReadFile("test_data/input/wfs.yaml")
	assert.NoError(t, err)
	inputWfs := pdoknlv3.WFS{}
	err = yaml.Unmarshal(input, &inputWfs)
	assert.NoError(t, err)
	warnings := []string{}
	allErrs := field.ErrorList{}
	pdoknlv3.ValidateWFS(&inputWfs, &warnings, &allErrs)

	inputStruct, err := MapWFSToMapfileGeneratorInput(&inputWfs, ownerInfo)
	assert.NoError(t, err)
	expected, err := readExpectedWFS("wfs.json")
	assert.NoError(t, err)

	diff := cmp.Diff(expected, inputStruct)
	assert.Equal(t, diff, "", "%s", diff)
}

func TestGetConfigForWMSWithNoGroupLayers(t *testing.T) {
	testWMS(t, "wms_groupless")
}

func TestGetConfigForWMSWithGroupLayers(t *testing.T) {
	testWMS(t, "wms_group")
}

func TestGetConfigForWMSWithGroupLayersAndTopGroupLayer(t *testing.T) {
	testWMS(t, "wms_group_and_toplayer")
}

func TestGetConfigForTifWMS(t *testing.T) {
	testWMS(t, "wms_tif")
}

func TestGetConfigForPostgisWMS(t *testing.T) {
	testWMS(t, "wms_postgis")
}

func testWMS(t *testing.T, filenameWithoutExt string) {
	pdoknlv3.SetHost("https://service.pdok.nl")
	ownerInfo := &smoothoperatorv1.OwnerInfo{
		Spec: smoothoperatorv1.OwnerInfoSpec{
			NamespaceTemplate: smoothoperatorutils.Pointer("http://{{prefix}}.geonovum.nl"),
		},
	}

	input, err := os.ReadFile("test_data/input/" + filenameWithoutExt + ".yaml")
	assert.NoError(t, err)
	v2wms := &v2beta1.WMS{}
	err = yaml.Unmarshal(input, v2wms)
	assert.NoError(t, err)
	var wms pdoknlv3.WMS
	err = v2wms.ToV3(&wms)
	assert.NoError(t, err)

	inputStruct, err := MapWMSToMapfileGeneratorInput(&wms, ownerInfo)
	assert.NoError(t, err)
	expected, err := readExpectedWMS(filenameWithoutExt + ".json")
	assert.NoError(t, err)

	diff := cmp.Diff(expected, inputStruct)
	assert.Equal(t, diff == "", true, "%s", diff)
}

func readExpectedWMS(filename string) (WMSInput, error) {
	bytes, err := os.ReadFile("test_data/expected/" + filename)
	if err != nil {
		return WMSInput{}, err
	}

	expected := WMSInput{}
	err = json.Unmarshal(bytes, &expected)

	return expected, err
}

func readExpectedWFS(filename string) (WFSInput, error) {
	bytes, err := os.ReadFile("test_data/expected/" + filename)
	if err != nil {
		return WFSInput{}, err
	}

	expected := WFSInput{}
	err = json.Unmarshal(bytes, &expected)

	return expected, err
}
