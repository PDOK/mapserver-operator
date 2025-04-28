package capabilitiesgenerator

import (
	"fmt"
	"gopkg.in/yaml.v3"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

func GetCapabilitiesGeneratorInitContainer[O pdoknlv3.WMSWFS](_ O, image string) (*corev1.Container, error) {
	initContainer := corev1.Container{
		Name:            "capabilities-generator",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name:  "SERVICECONFIG",
				Value: "/input/input.yaml",
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "data", MountPath: "/var/www", ReadOnly: false},
			{Name: mapserver.ConfigMapCapabilitiesGeneratorVolumeName, MountPath: "/input", ReadOnly: true},
		},
	}
	return &initContainer, nil
}

func GetInput[W pdoknlv3.WMSWFS](webservice W, ownerInfo *smoothoperatorv1.OwnerInfo) (input string, err error) {
	switch any(webservice).(type) {
	case *pdoknlv3.WFS:
		if WFS, ok := any(webservice).(*pdoknlv3.WFS); ok {
			return createInputForWFS(WFS, ownerInfo)
		}
	case *pdoknlv3.WMS:
		if WMS, ok := any(webservice).(*pdoknlv3.WMS); ok {
			return createInputForWMS(WMS, ownerInfo)
		}
	default:
		return "", fmt.Errorf("unexpected input, webservice should be of type WFS or WMS, webservice: %v", webservice)
	}
	return "", fmt.Errorf("unexpected input, webservice should be of type WFS or WMS, webservice: %v", webservice)
}

func createInputForWFS(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	input, err := MapWFSToCapabilitiesGeneratorInput(wfs, ownerInfo)
	if err != nil {
		return "", err
	}
	yamlInput, err := yaml.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal the capabilities generator input to yaml: %w", err)
	}

	return string(yamlInput), nil
}

func createInputForWMS(wms *pdoknlv3.WMS, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	input, err := MapWMSToCapabilitiesGeneratorInput(wms, ownerInfo)
	if err != nil {
		return "", err
	}
	yamlInput, err := yaml.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal the capabilities generator input to yaml: %w", err)
	}

	return string(yamlInput), nil
}
