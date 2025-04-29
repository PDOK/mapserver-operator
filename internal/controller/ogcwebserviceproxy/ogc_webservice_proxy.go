package ogcwebserviceproxy

import (
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func GetOgcWebserviceProxyContainer(wms *pdoknlv3.WMS, image string) (*corev1.Container, error) {
	container := corev1.Container{
		Name:            "ogc-webservice-proxy",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports:           []corev1.ContainerPort{{ContainerPort: 9111}},
		Command:         getCommand(wms),
		VolumeMounts: []corev1.VolumeMount{
			{Name: mapserver.ConfigMapOgcWebserviceProxyVolumeName, MountPath: "/input", ReadOnly: true},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("200M"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("0.05"),
			},
		},
	}
	return &container, nil
}

func getCommand(wms *pdoknlv3.WMS) []string {
	command := []string{
		"/ogc-webservice-proxy",
		"-h=http://127.0.0.1/",
		"-t=wms",
		"-s=/input/service-config.yaml",
	}

	if wms.Options() != nil && *wms.Options().ValidateRequests {
		command = append(command, "-v")
	}
	if wms.Options() != nil && *wms.Options().RewriteGroupToDataLayers {
		command = append(command, "-r")
	}
	command = append(command, "-d=15")
	return command

}

func GetConfig(wms *pdoknlv3.WMS) (config string, err error) {
	input, err := MapWMSToOgcWebserviceProxyConfig(wms)
	if err != nil {
		return "", err
	}

	yamlConfig, err := yaml.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(yamlConfig), nil
}

func MapWMSToOgcWebserviceProxyConfig(wms *pdoknlv3.WMS) (config Config, err error) {
	config.GroupLayers = make(map[string][]string)
	for _, layer := range wms.Spec.Service.Layer.GetAllLayers() {
		// TODO Should there be a distinction between grouplayers and the toplayer?
		if layer.IsGroupLayer() {
			var dataLayers []string
			for _, childLayer := range *layer.Layers {
				if childLayer.IsDataLayer() {
					dataLayers = append(dataLayers, *childLayer.Name)
				}
			}
			if len(dataLayers) > 0 {
				config.GroupLayers[smoothoperatorutils.PointerVal(layer.Name, "")] = dataLayers
			}
		}
	}
	return
}

type Config struct {
	GroupLayers map[string][]string `yaml:"grouplayers"`
}
