package legendgenerator

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/pdok/mapserver-operator/api/v2beta1"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func test(t *testing.T, name string) {
	input, err := os.ReadFile("test_data/input/" + name + ".yaml")
	assert.NoError(t, err)
	var v2wms v2beta1.WMS
	err = yaml.Unmarshal(input, &v2wms)
	assert.NoError(t, err)
	var wms pdoknlv3.WMS
	err = v2wms.ToV3(&wms)
	assert.NoError(t, err)

	expected, err := os.ReadFile("test_data/expected/" + name + ".yaml")
	assert.NoError(t, err)

	expectedMap := make(map[string]string)
	err = yaml.Unmarshal(expected, &expectedMap)
	assert.NoError(t, err)

	diff := cmp.Diff(expectedMap, GetConfigMapData(&wms))
	assert.Equal(t, diff, "", "diff in %s, -want +got: %s", name, diff)
}

func TestGetConfigMapDataNoLegendFix(t *testing.T) {
	test(t, "no-legend-fix")
}

func TestGetConfigMapDataLegendFix(t *testing.T) {
	test(t, "no-legend-fix")
}
