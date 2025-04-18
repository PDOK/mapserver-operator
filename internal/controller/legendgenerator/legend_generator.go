package legendgenerator

import (
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	corev1 "k8s.io/api/core/v1"
)

func GetLegendGeneratorInitContainer(wms *pdoknlv3.WMS, image string, srvDir string) (*corev1.Container, error) {
	initContainer := corev1.Container{
		Name:            "legend-generator",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             []corev1.EnvVar{mapserver.GetMapfileEnvVar(wms)},
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
			{Name: "base", MountPath: srvDir + "/data", ReadOnly: false},
			getDataVolumeMount(),
			getConfigVolumeMount(),
		},
	}

	if wms.Spec.Service.Mapfile != nil {
		volumeMount := corev1.VolumeMount{
			Name:      "mapfile",
			MountPath: "/srv/data/config/mapfile",
		}
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, volumeMount)
	}

	return &initContainer, nil
}

func GetLegendFixerInitContainer(image string) *corev1.Container {
	return &corev1.Container{
		Name:            "legend-fixer",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/bin/bash",
			"/input/legend-fixer.sh",
		},
		VolumeMounts: []corev1.VolumeMount{
			getDataVolumeMount(),
			getConfigVolumeMount(),
		},
	}
}

func GetConfigMapData(wms *pdoknlv3.WMS) map[string]string {
	data := map[string]string{
		"default_mapserver.conf": defaultMapserverConf,
	}

	addLayerInput(wms, data)
	if wms.Spec.Options.RewriteGroupToDataLayers != nil && *wms.Spec.Options.RewriteGroupToDataLayers {
		addLegendFixerConfig(wms, data)
	}
	return data
}

func getDataVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: "data", MountPath: "/var/www", ReadOnly: false}
}

func getConfigVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{Name: mapserver.ConfigMapLegendGeneratorVolumeName, MountPath: "/input", ReadOnly: true}
}
