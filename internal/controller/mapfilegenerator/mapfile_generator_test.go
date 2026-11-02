package mapfilegenerator

import (
	"encoding/json"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/google/go-cmp/cmp"

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

func TestWMSLayerScaleInheritance(t *testing.T) {
	for _, testcase := range []struct {
		name        string
		topMax      *string
		groupMax    *string
		childMax    *string
		groupMin    *string
		childMin    *string
		expectedMax *string
		expectedMin *string
	}{
		{
			name: "inherit group scales", topMax: smoothoperatorutils.Pointer("15000000"),
			groupMax: smoothoperatorutils.Pointer("50000"), groupMin: smoothoperatorutils.Pointer("2"),
			expectedMax: smoothoperatorutils.Pointer("50000"), expectedMin: smoothoperatorutils.Pointer("2"),
		},
		{
			name: "inherit top scales through group", topMax: smoothoperatorutils.Pointer("15000000"),
			expectedMax: smoothoperatorutils.Pointer("15000000"), expectedMin: smoothoperatorutils.Pointer("1"),
		},
		{
			name: "preserve child overrides", topMax: smoothoperatorutils.Pointer("15000000"),
			groupMax: smoothoperatorutils.Pointer("50000"), groupMin: smoothoperatorutils.Pointer("2"),
			childMax: smoothoperatorutils.Pointer("10000"), childMin: smoothoperatorutils.Pointer("3"),
			expectedMax: smoothoperatorutils.Pointer("10000"), expectedMin: smoothoperatorutils.Pointer("3"),
		},
		{
			name: "inherit scales independently", topMax: smoothoperatorutils.Pointer("15000000"),
			groupMin: smoothoperatorutils.Pointer("2"), childMax: smoothoperatorutils.Pointer("10000"),
			expectedMax: smoothoperatorutils.Pointer("10000"), expectedMin: smoothoperatorutils.Pointer("2"),
		},
		{
			name: "leave maximum unset without ancestors", expectedMin: smoothoperatorutils.Pointer("1"),
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			child := pdoknlv3.Layer{
				Name:                smoothoperatorutils.Pointer("child"),
				MaxScaleDenominator: testcase.childMax, MinScaleDenominator: testcase.childMin,
			}
			var wms pdoknlv3.WMS
			wms.Spec.Service.Layer = pdoknlv3.Layer{
				Name:                smoothoperatorutils.Pointer("top"),
				MaxScaleDenominator: testcase.topMax, MinScaleDenominator: smoothoperatorutils.Pointer("1"),
				Layers: []pdoknlv3.Layer{{
					Name:                smoothoperatorutils.Pointer("group"),
					MaxScaleDenominator: testcase.groupMax, MinScaleDenominator: testcase.groupMin,
					Layers: []pdoknlv3.Layer{child},
				}},
			}
			original := wms.DeepCopy()
			result := getWMSLayer(child, "", &wms)
			assert.Equal(t, testcase.expectedMax, result.MaxScale)
			assert.Equal(t, testcase.expectedMin, result.MinScale)
			assert.Equal(t, original, &wms)
		})
	}
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
	var wms pdoknlv3.WMS
	err = yaml.Unmarshal(input, &wms)
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
