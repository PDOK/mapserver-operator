package utils

import corev1 "k8s.io/api/core/v1"

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
