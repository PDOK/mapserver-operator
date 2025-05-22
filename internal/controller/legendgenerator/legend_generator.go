package legendgenerator

import (
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/constants"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	"github.com/pdok/mapserver-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
)

func GetLegendGeneratorInitContainer(wms *pdoknlv3.WMS, images types.Images) (*corev1.Container, error) {
	initContainer := corev1.Container{
		Name:            constants.LegendGeneratorName,
		Image:           images.MapserverImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name:  "MAPSERVER_CONFIG_FILE",
				Value: "/srv/mapserver/config/default_mapserver.conf",
			},
			mapserver.GetMapfileEnvVar(wms),
		},
		Command: []string{
			"bash",
			"-c",
			`set -eu;
			exit_code=0;
			cat /input/input | xargs -n 2 echo | while read layer style; do
			echo Generating legend for layer: $layer, style: $style;
			mkdir -p /var/www/legend/$layer;
			mapserv -nh 'QUERY_STRING=SERVICE=WMS&language=dut&version=1.3.0&service=WMS&request=GetLegendGraphic&sld_version=1.1.0&layer='$layer'&format=image/png&STYLE='$style'' > /var/www/legend/$layer/${style}.png;
			magic_bytes=$(head -c 4 /var/www/legend/$layer/${style}.png | tail -c 3);
			if [[ $magic_bytes != 'PNG' ]]; then
			echo [4T2O9] file /var/www/legend/$layer/${style}.png appears to not be a png file;
			exit_code=1;
			fi;
			done;
			exit $exit_code;
`,
		},
		VolumeMounts: []corev1.VolumeMount{
			utils.GetBaseVolumeMount(),
			utils.GetDataVolumeMount(),
			{Name: constants.MapserverName, MountPath: "/srv/mapserver/config/default_mapserver.conf", SubPath: "default_mapserver.conf"},
		},
	}

	if wms.Spec.Service.Mapfile != nil {
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, utils.GetMapfileVolumeMount())
	}

	// Adding config volumemount here to get the same order as in the old ansible operator
	initContainer.VolumeMounts = append(initContainer.VolumeMounts, utils.GetConfigVolumeMount(constants.ConfigMapLegendGeneratorVolumeName))

	return &initContainer, nil
}

func GetLegendFixerInitContainer(images types.Images) *corev1.Container {
	return &corev1.Container{
		Name:            constants.LegendFixerName,
		Image:           images.MultitoolImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/bin/bash",
			"/input/legend-fixer.sh",
		},
		VolumeMounts: []corev1.VolumeMount{
			utils.GetDataVolumeMount(),
			utils.GetConfigVolumeMount(constants.ConfigMapLegendGeneratorVolumeName),
		},
	}
}

func GetConfigMapData(wms *pdoknlv3.WMS) map[string]string {
	data := map[string]string{
		"default_mapserver.conf": defaultMapserverConf,
	}

	addLayerInput(wms, data)
	if wms.Options().RewriteGroupToDataLayers {
		addLegendFixerConfig(wms, data)
	}
	return data
}
