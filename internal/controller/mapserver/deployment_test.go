package mapserver

import (
	"testing"

	"github.com/pdok/mapserver-operator/internal/controller/utils"

	"github.com/pdok/mapserver-operator/api/v2beta1"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"

	_ "embed"
)

//go:embed test_data/expected_volumemounts.yaml
var expectedVolumeMountsYaml []byte

func TestGetVolumeMountsForDeployment(t *testing.T) {
	var wfs = getV3()
	pdoknlv3.SetHost("https://service.pdok.nl")
	result := GetVolumeMountsForDeployment(wfs)

	var expectedVolumeMounts struct{ VolumeMounts []corev1.VolumeMount }
	err := yaml.Unmarshal(expectedVolumeMountsYaml, &expectedVolumeMounts)
	assert.NoError(t, err)
	assert.Equal(t, expectedVolumeMounts.VolumeMounts, result)
}

//go:embed test_data/expected_envvars.yaml
var expectedEnvVarsYaml []byte

func TestGetEnvVarsForDeployment(t *testing.T) {
	var wfs = getV3()
	pdoknlv3.SetHost("https://service.pdok.nl")
	result := GetEnvVarsForDeployment(wfs, "blobs-secret")
	var expectedEnvVars struct{ EnvVars []corev1.EnvVar }
	err := yaml.Unmarshal(expectedEnvVarsYaml, &expectedEnvVars)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnvVars.EnvVars, result)
}

func TestGetResourcesForDeployment(t *testing.T) {
	var wfs = getV3()
	pdoknlv3.SetHost("https://service.pdok.nl")
	result := GetResourcesForDeployment(wfs)

	expectedLimits := corev1.ResourceList{}
	expectedRequest := corev1.ResourceList{}

	expectedLimits[corev1.ResourceMemory] = resource.MustParse("800M")
	expectedLimits[corev1.ResourceEphemeralStorage] = resource.MustParse("505Mi")

	expectedRequest[corev1.ResourceCPU] = resource.MustParse("0.15")
	expectedRequest[corev1.ResourceEphemeralStorage] = resource.MustParse("255Mi")

	var expected = corev1.ResourceRequirements{
		Limits:   expectedLimits,
		Requests: expectedRequest,
		Claims:   nil,
	}

	assert.Equal(t, expected, result)
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
	livenessResult, readinessResult, startupResult, err := GetProbesForDeployment(wfs)
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

func TestGetVolumesForDeployment(t *testing.T) {
	var wfs = getV3()
	wfs.Spec.Options.PrefetchData = false
	pdoknlv3.SetHost("https://service.pdok.nl")

	hashedConfigMapNames := types.HashedConfigMapNames{
		ConfigMap:             "rws-nwbwegen-v1-0-wfs-mapserver-bb59c7f4f4",
		BlobDownload:          "2",
		MapfileGenerator:      "rws-nwbwegen-v1-0-wfs-mapfile-generator-bbbtd999dh",
		CapabilitiesGenerator: "rws-nwbwegen-v1-0-wfs-capabilities-generator-6m4mfkgb5d",
		OgcWebserviceProxy:    "3",
		LegendGenerator:       "4",
		FeatureInfoGenerator:  "5",
	}
	result := GetVolumesForDeployment(wfs, hashedConfigMapNames)

	expected := []corev1.Volume{
		{Name: "base", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: utils.MapserverName, VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "rws-nwbwegen-v1-0-wfs-mapserver-bb59c7f4f4"}, DefaultMode: smoothoperatorutils.Pointer(int32(420))}}},
		{Name: "capabilities-generator-config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "rws-nwbwegen-v1-0-wfs-capabilities-generator-6m4mfkgb5d"}, DefaultMode: smoothoperatorutils.Pointer(int32(420))}}},
		{Name: "mapfile-generator-config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "rws-nwbwegen-v1-0-wfs-mapfile-generator-bbbtd999dh"}, DefaultMode: smoothoperatorutils.Pointer(int32(420))}}},
	}

	assert.Equal(t, expected, result)
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
