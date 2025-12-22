package mapserver

import (
	"testing"

	"github.com/pdok/mapserver-operator/api/v2beta1"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	_ "embed"
)

//go:embed test_data/expected_volumemounts.yaml
var expectedVolumeMountsYaml []byte

func TestGetVolumeMounts(t *testing.T) {
	pdoknlv3.SetHost("https://service.pdok.nl")
	result := getVolumeMounts(false)

	var expectedVolumeMounts struct{ VolumeMounts []corev1.VolumeMount }
	err := yaml.Unmarshal(expectedVolumeMountsYaml, &expectedVolumeMounts)
	assert.NoError(t, err)
	assert.Equal(t, expectedVolumeMounts.VolumeMounts, result)
}

//go:embed test_data/expected_livenessprobe.yaml
var expectedLivenessProbe []byte

//go:embed test_data/expected_readinessprobe.yaml
var expectedReadinessProbe []byte

//go:embed test_data/expected_startupprobe.yaml
var expectedStartupProbe []byte

func TestGetProbesForDeployment(t *testing.T) {
	var wfs = getV3()
	pdoknlv3.SetHost("https://service.pdok.nl")
	livenessResult, readinessResult, startupResult, err := getProbes(wfs)
	assert.NoError(t, err)

	var expectedLiveness corev1.Probe
	var expectedReadiness corev1.Probe
	var expectedStartup corev1.Probe
	err = yaml.Unmarshal(expectedLivenessProbe, &expectedLiveness)
	assert.NoError(t, err)
	err = yaml.Unmarshal(expectedReadinessProbe, &expectedReadiness)
	assert.NoError(t, err)
	err = yaml.Unmarshal(expectedStartupProbe, &expectedStartup)
	assert.NoError(t, err)
	assert.Equal(t, &expectedLiveness, livenessResult)
	assert.Equal(t, &expectedReadiness, readinessResult)
	assert.Equal(t, &expectedStartup, startupResult)
}

//go:embed test_data/v2_input.yaml
var v2Input []byte

func getV3() *pdoknlv3.WFS {
	var v2wfs v2beta1.WFS
	err := yaml.Unmarshal(v2Input, &v2wfs)
	if err != nil {
		panic(err)
	}
	var wfs pdoknlv3.WFS
	_ = v2wfs.ToV3(&wfs)
	return &wfs
}
