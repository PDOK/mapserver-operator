package utils

import (
	//nolint:gosec
	"crypto/sha1"
	"encoding/hex"
	"io"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	corev1 "k8s.io/api/core/v1"
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
	return corev1.VolumeMount{Name: constants.BaseVolumeName, MountPath: "/srv/data", ReadOnly: false}
}

func GetDataVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: constants.DataVolumeName, MountPath: "/var/www", ReadOnly: false}
}

func GetConfigVolumeMount(volumeName string) corev1.VolumeMount {
	return corev1.VolumeMount{Name: volumeName, MountPath: "/input", ReadOnly: true}
}

func GetMapfileVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: constants.ConfigMapCustomMapfileVolumeName, MountPath: "/srv/data/config/mapfile"}
}

func Sha1Hash(v string) string {
	//nolint:gosec
	s := sha1.New()
	_, _ = io.WriteString(s, v)

	return hex.EncodeToString(s.Sum(nil))
}
