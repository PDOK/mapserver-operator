package featureinfogenerator

import (
	"encoding/json"
	"fmt"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	"github.com/pdok/mapserver-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
)

func GetFeatureinfoGeneratorInitContainer(images types.Images) (*corev1.Container, error) {
	initContainer := corev1.Container{
		Name:            constants.FeatureinfoGeneratorName,
		Image:           images.FeatureinfoGeneratorImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{constants.FeatureinfoGeneratorName},
		Args: []string{
			"--input-path",
			"/input/input.json",
			"--dest-folder",
			constants.HTMLTemplatesPath,
			"--file-name",
			"feature-info",
		},
		VolumeMounts: []corev1.VolumeMount{
			utils.GetBaseVolumeMount(false),
			utils.GetConfigVolumeMount(constants.ConfigMapFeatureinfoGeneratorVolumeName),
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
