package mapserver

import (
	"errors"
	"strings"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	"github.com/pdok/mapserver-operator/internal/controller/utils"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/static"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const mimeTextXML = "text/xml"

func GetMapserverContainer[O pdoknlv3.WMSWFS](obj O, images types.Images, blobsSecretName string) (*corev1.Container, error) {
	livenessProbe, readinessProbe, startupProbe, err := getProbes(obj)
	if err != nil {
		return nil, err
	}

	container := corev1.Container{
		Name:            constants.MapserverName,
		Image:           images.MapserverImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports:           []corev1.ContainerPort{{ContainerPort: constants.MapserverPortNr, Protocol: corev1.ProtocolTCP}},
		Env: []corev1.EnvVar{
			{
				Name:  "SERVICE_TYPE",
				Value: string(obj.Type()),
			},
			{
				Name:  "MAPSERVER_CONFIG_FILE",
				Value: "/srv/mapserver/config/default_mapserver.conf",
			},
			GetMapfileEnvVar(obj),
			{
				Name: "AZURE_STORAGE_CONNECTION_STRING",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: blobsSecretName},
						Key:                  "AZURE_STORAGE_CONNECTION_STRING",
					},
				},
			},
		},
		VolumeMounts: getVolumeMounts(obj.Mapfile() != nil),
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("800M"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("0.15"),
			},
		},
		Lifecycle:      &corev1.Lifecycle{PreStop: &corev1.LifecycleHandler{Exec: &corev1.ExecAction{Command: []string{"sleep", "15"}}}},
		StartupProbe:   startupProbe,
		ReadinessProbe: readinessProbe,
		LivenessProbe:  livenessProbe,
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS && !obj.Options().DisableWebserviceProxy {
		container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse("0.1")
	}

	return &container, nil
}

func getVolumeMounts(customMapfile bool) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		utils.GetBaseVolumeMount(),
		utils.GetDataVolumeMount(),
	}

	staticFiles, _ := static.GetStaticFiles()
	for _, name := range staticFiles {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      constants.MapserverName,
			MountPath: "/srv/mapserver/config/" + name,
			SubPath:   name,
		})
	}
	if customMapfile {
		volumeMounts = append(volumeMounts, utils.GetMapfileVolumeMount())
	}

	return volumeMounts
}

func GetMapfileEnvVar[O pdoknlv3.WMSWFS](obj O) corev1.EnvVar {
	mapFileName := "service.map"
	if obj.Mapfile() != nil {
		mapFileName = obj.Mapfile().ConfigMapKeyRef.Key
	}

	return corev1.EnvVar{
		Name:  "MS_MAPFILE",
		Value: "/srv/data/config/mapfile/" + mapFileName,
	}
}

func getProbes[O pdoknlv3.WMSWFS](obj O) (livenessProbe *corev1.Probe, readinessProbe *corev1.Probe, startupProbe *corev1.Probe, err error) {
	livenessProbe = getLivenessProbe(obj)
	switch obj.Type() {
	case pdoknlv3.ServiceTypeWFS:
		wfs, _ := any(obj).(*pdoknlv3.WFS)
		readinessProbe, err = getReadinessProbeForWFS(wfs)
		if err != nil {
			return nil, nil, nil, err
		}
		startupProbe, err = getStartupProbeForWFS(wfs)
		if err != nil {
			return nil, nil, nil, err
		}
	case pdoknlv3.ServiceTypeWMS:
		wms, _ := any(obj).(*pdoknlv3.WMS)
		readinessProbe, err = getReadinessProbeForWMS(wms)
		if err != nil {
			return nil, nil, nil, err
		}
		startupProbe, err = getStartupProbeForWMS(wms)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return
}

func getLivenessProbe[O pdoknlv3.WMSWFS](obj O) *corev1.Probe {
	queryString := "SERVICE=" + string(obj.Type()) + "&request=GetCapabilities"
	return getProbe(queryString, mimeTextXML)
}

func getReadinessProbeForWFS(wfs *pdoknlv3.WFS) (*corev1.Probe, error) {
	queryString, mime, err := wfs.ReadinessQueryString()
	if err != nil {
		return nil, err
	}
	return getProbe(queryString, mime), nil
}

func getReadinessProbeForWMS(wms *pdoknlv3.WMS) (*corev1.Probe, error) {
	queryString, mime, err := wms.ReadinessQueryString()
	if err != nil {
		return nil, err
	}

	return getProbe(queryString, mime), nil
}

func getStartupProbeForWFS(wfs *pdoknlv3.WFS) (*corev1.Probe, error) {
	if hc := wfs.Spec.HealthCheck; hc != nil {
		return getProbe(hc.Querystring, hc.Mimetype), nil
	}

	var typeNames []string
	for _, ft := range wfs.Spec.Service.FeatureTypes {
		typeNames = append(typeNames, ft.Name)
	}
	if len(typeNames) == 0 {
		return nil, errors.New("cannot get startup probe for WFS, featuretypes could not be found")
	}

	queryString := "SERVICE=WFS&VERSION=2.0.0&REQUEST=GetFeature&TYPENAMES=" + strings.Join(typeNames, ",") + "&STARTINDEX=0&COUNT=1"
	return getProbe(queryString, mimeTextXML), nil
}

func getStartupProbeForWMS(wms *pdoknlv3.WMS) (*corev1.Probe, error) {
	if hc := wms.Spec.HealthCheck; hc != nil && hc.Querystring != nil {
		return getProbe(*hc.Querystring, *hc.Mimetype), nil
	}

	var layerNames []string
	for _, layer := range wms.Spec.Service.GetAllLayers() {
		if layer.Name != nil {
			layerNames = append(layerNames, *layer.Name)
		}

	}
	if len(layerNames) == 0 {
		return nil, errors.New("cannot get startup probe for WMS, layers could not be found")
	}

	queryString := "SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&BBOX=" + wms.HealthCheckBBox() + "&CRS=EPSG:28992&WIDTH=100&HEIGHT=100&LAYERS=" + strings.Join(layerNames, ",") + "&STYLES=&FORMAT=image/png"
	mimeType := "image/png"
	return getProbe(queryString, mimeType), nil
}

func getProbe(queryString string, mimeType string) *corev1.Probe {
	probeCmd := "wget -SO- -T 10 -t 2 'http://127.0.0.1:80/mapserver?" + queryString + "' 2>&1 | egrep -aiA10 'HTTP/1.1 200' | egrep -i 'Content-Type: " + mimeType + "'"
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{
			Command: []string{
				"/bin/sh",
				"-c",
				probeCmd,
			},
		}},
		SuccessThreshold:    1,
		FailureThreshold:    3,
		InitialDelaySeconds: 20,
		PeriodSeconds:       10,
		TimeoutSeconds:      10,
	}
}
