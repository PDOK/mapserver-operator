package mapfilegenerator

import (
	"encoding/json"
	"fmt"
	"strings"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	"github.com/pdok/mapserver-operator/internal/controller/utils"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

func GetMapfileGeneratorInitContainer[O pdoknlv3.WMSWFS](obj O, image, postgisConfigName, postgisSecretName, srvDir string) (*corev1.Container, error) {
	initContainer := corev1.Container{
		Name:            "mapfile-generator",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"generate-mapfile"},
		Args: []string{
			"--not-include",
			strings.ToLower(string(obj.Type())),
			"/input/input.json",
			"/srv/data/config/mapfile",
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "base", MountPath: srvDir + "/data", ReadOnly: false},
			{Name: mapserver.ConfigMapMapfileGeneratorVolumeName, MountPath: "/input", ReadOnly: true},
		},
	}

	// Additional mapfile-generator configuration
	if obj.HasPostgisData() {
		initContainer.EnvFrom = []corev1.EnvFromSource{
			// Todo add this ConfigMap
			utils.NewEnvFromSource(utils.EnvFromSourceTypeConfigMap, postgisConfigName),
			// Todo add this Secret
			utils.NewEnvFromSource(utils.EnvFromSourceTypeSecret, postgisSecretName),
		}
	}
	return &initContainer, nil
}

func GetConfig[W pdoknlv3.WMSWFS](webservice W, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	switch any(webservice).(type) {
	case *pdoknlv3.WFS:
		if WFS, ok := any(webservice).(*pdoknlv3.WFS); ok {
			return createConfigForWFS(WFS, ownerInfo)
		}
	case *pdoknlv3.WMS:
		if WMS, ok := any(webservice).(*pdoknlv3.WMS); ok {
			return createConfigForWMS(WMS, ownerInfo)
		}
	default:
		return "", fmt.Errorf("unexpected input, webservice should be of type WFS or WMS, webservice: %v", webservice)
	}
	return "", fmt.Errorf("unexpected input, webservice should be of type WFS or WMS, webservice: %v", webservice)
}

func createConfigForWFS(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	input, err := MapWFSToMapfileGeneratorInput(wfs, ownerInfo)
	if err != nil {
		return "", err
	}

	jsonConfig, err := json.MarshalIndent(input, "", "    ")
	if err != nil {
		return "", err
	}
	return string(jsonConfig), nil
}

func createConfigForWMS(wms *pdoknlv3.WMS, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	input, err := MapWMSToMapfileGeneratorInput(wms, ownerInfo)
	if err != nil {
		return "", err
	}

	jsonConfig, err := json.MarshalIndent(input, "", "    ")
	if err != nil {
		return "", err
	}
	return string(jsonConfig), nil
}
