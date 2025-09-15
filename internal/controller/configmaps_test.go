package controller

import (
	"os"
	"testing"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/constants"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestMapserverConfigMaps(t *testing.T) {
	wfsBytes, err := os.ReadFile("test_data/wfs/complete/input/wfs.yaml")
	assert.NoError(t, err)
	o := &pdoknlv3.WFS{}
	err = yaml.Unmarshal(wfsBytes, o)
	assert.NoError(t, err)
	generatedConfigMap := getBareConfigMap(o, constants.MapserverName)
	generatedConfigMap.Data = make(map[string]string)
	updateConfigMapWithStaticFiles(generatedConfigMap, o)

	expectedConfigMap := v1.ConfigMap{}
	expectedBytes, err := os.ReadFile("test_data/wfs/complete/expected/configmap-mapserver.yaml")
	err = yaml.Unmarshal(expectedBytes, &expectedConfigMap)
	assert.NoError(t, err)

	assert.Equal(t, expectedConfigMap.Data, generatedConfigMap.Data)
}
