package utils

import (
	//nolint:gosec
	"crypto/sha1"
	"encoding/hex"
	"io"

	corev1 "k8s.io/api/core/v1"
)

const (
	MapserverName             = "mapserver"
	OgcWebserviceProxyName    = "ogc-webservice-proxy"
	MapfileGeneratorName      = "mapfile-generator"
	CapabilitiesGeneratorName = "capabilities-generator"
	BlobDownloadName          = "blob-download"
	InitScriptsName           = "init-scripts"
	LegendGeneratorName       = "legend-generator"
	LegendFixerName           = "legend-fixer"
	FeatureinfoGeneratorName  = "featureinfo-generator"

	configSuffix                             = "-config"
	ConfigMapMapfileGeneratorVolumeName      = MapfileGeneratorName + configSuffix
	ConfigMapStylingFilesVolumeName          = "styling-files"
	ConfigMapCapabilitiesGeneratorVolumeName = CapabilitiesGeneratorName + configSuffix
	ConfigMapOgcWebserviceProxyVolumeName    = OgcWebserviceProxyName + configSuffix
	ConfigMapLegendGeneratorVolumeName       = LegendGeneratorName + configSuffix
	ConfigMapFeatureinfoGeneratorVolumeName  = FeatureinfoGeneratorName + configSuffix

	HTMLTemplatesPath = "/srv/data/config/templates"
)

type EnvFromSourceType string

const (
	EnvFromSourceTypeConfigMap EnvFromSourceType = "configMap"
	EnvFromSourceTypeSecret    EnvFromSourceType = "secret"
)

func NewEnvFromSource(t EnvFromSourceType, name string) corev1.EnvFromSource {
	switch t {
	case EnvFromSourceTypeConfigMap:
		return corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		}
	case EnvFromSourceTypeSecret:
		return corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		}
	default:
		return corev1.EnvFromSource{}
	}
}

func GetBaseVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: "base", MountPath: "/srv/data", ReadOnly: false}
}

func GetDataVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: "data", MountPath: "/var/www", ReadOnly: false}
}

func GetConfigVolumeMount(volumeName string) corev1.VolumeMount {
	return corev1.VolumeMount{Name: volumeName, MountPath: "/input", ReadOnly: true}
}

func Sha1Hash(v string) string {
	//nolint:gosec
	s := sha1.New()
	_, _ = io.WriteString(s, v)

	return hex.EncodeToString(s.Sum(nil))
}
