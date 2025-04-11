package featureinfogenerator

import (
	"encoding/json"
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	corev1 "k8s.io/api/core/v1"
)

const (
	htmlTemplatesPath                       = "/srv/data/config/templates"
	ConfigMapFeatureinfoGeneratorVolumeName = "featureinfo-generator-config"
)

func GetFeatureinfoGeneratorInitContainer(image string, srvDir string) (*corev1.Container, error) {
	initContainer := corev1.Container{
		Name:            "featureinfo-generator",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"featureinfo-generator"},
		Args: []string{
			"--input-path",
			"/input/input.json",
			"--dest-folder",
			htmlTemplatesPath,
			"--file-name",
			"feature-info",
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "base", MountPath: srvDir + "/data", ReadOnly: false},
			{Name: ConfigMapFeatureinfoGeneratorVolumeName, MountPath: "/input", ReadOnly: true},
		},
	}

	return &initContainer, nil
}

func GetInput(wms *pdoknlv3.WMS) (string, error) {
	input, err := MapWMSToFeatureinfoGeneratorInput(wms)
	if err != nil {
		return "", err
	}
	jsonInput, err := json.MarshalIndent(input, "", "    ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal the featureinfo generator input to json: %w", err)
	}

	return string(jsonInput), nil
}
