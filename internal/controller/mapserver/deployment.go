package mapserver

import (
	"errors"
	"os"
	"strings"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	"github.com/pdok/mapserver-operator/internal/controller/utils"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	"github.com/pdok/mapserver-operator/internal/controller/static"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	MapserverPortNr int32 = 80

	// TODO How should we determine this boundingbox?
	// healthCheckBbox = "190061.4619730016857,462435.5987861062749,202917.7508707302331,473761.6884966178914"

	mimeTextXML = "text/xml"
)

func GetMapserverContainer[O pdoknlv3.WMSWFS](obj O, images types.Images, blobsSecretName string) (*corev1.Container, error) {
	livenessProbe, readinessProbe, startupProbe, err := GetProbesForDeployment(obj)
	if err != nil {
		return nil, err
	}

	container := corev1.Container{
		Name:            constants.MapserverName,
		Image:           images.MapserverImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports:           []corev1.ContainerPort{{ContainerPort: MapserverPortNr}},
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

// TODO fix linting (funlen)
//
//nolint:funlen
func GetVolumesForDeployment[O pdoknlv3.WMSWFS](obj O, configMapNames types.HashedConfigMapNames) []corev1.Volume {
	baseVolume := corev1.Volume{Name: constants.BaseVolumeName}
	if use, size := mapperutils.UseEphemeralVolume(obj); use {
		baseVolume.Ephemeral = &corev1.EphemeralVolumeSource{
			VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: *size,
						},
					},
				},
			},
		}

		if value, set := os.LookupEnv("STORAGE_CLASS_NAME"); set {
			baseVolume.Ephemeral.VolumeClaimTemplate.Spec.StorageClassName = &value
		}
	} else {
		baseVolume.VolumeSource = corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		}
	}

	newVolumeSource := func(name string) corev1.VolumeSource {
		return corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				DefaultMode: smoothoperatorutils.Pointer(int32(0644)),
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		}
	}

	volumes := []corev1.Volume{
		baseVolume,
		{
			Name: constants.DataVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name:         constants.MapserverName,
			VolumeSource: newVolumeSource(configMapNames.Mapserver),
		},
	}

	if mapfile := obj.Mapfile(); mapfile != nil {
		volumes = append(volumes, corev1.Volume{
			Name:         constants.ConfigMapCustomMapfileVolumeName,
			VolumeSource: newVolumeSource(mapfile.ConfigMapKeyRef.Name),
		})
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS && obj.Options().UseWebserviceProxy() {
		volumes = append(volumes, corev1.Volume{
			Name:         constants.ConfigMapOgcWebserviceProxyVolumeName,
			VolumeSource: newVolumeSource(configMapNames.OgcWebserviceProxy),
		})
	}

	if obj.Options().PrefetchData {
		vol := newVolumeSource(configMapNames.InitScripts)
		vol.ConfigMap.DefaultMode = smoothoperatorutils.Pointer(int32(0777))
		volumes = append(volumes, corev1.Volume{
			Name:         constants.InitScriptsName,
			VolumeSource: vol,
		})
	}

	// Add capabilitiesgenerator config here to get the same order as the ansible operator
	// Needed to compare deployments from the ansible operator and this one
	volumes = append(volumes, corev1.Volume{
		Name:         constants.ConfigMapCapabilitiesGeneratorVolumeName,
		VolumeSource: newVolumeSource(configMapNames.CapabilitiesGenerator),
	})

	var stylingFilesVolume *corev1.Volume
	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		lgVolume := corev1.Volume{
			Name:         constants.ConfigMapLegendGeneratorVolumeName,
			VolumeSource: newVolumeSource(configMapNames.LegendGenerator),
		}
		figVolume := corev1.Volume{
			Name:         constants.ConfigMapFeatureinfoGeneratorVolumeName,
			VolumeSource: newVolumeSource(configMapNames.FeatureInfoGenerator),
		}

		wms, _ := any(obj).(*pdoknlv3.WMS)
		stylingFilesVolumeProjections := []corev1.VolumeProjection{}
		if wms.Spec.Service.StylingAssets != nil && wms.Spec.Service.StylingAssets.ConfigMapRefs != nil {
			for _, cf := range wms.Spec.Service.StylingAssets.ConfigMapRefs {
				stylingFilesVolumeProjections = append(stylingFilesVolumeProjections, corev1.VolumeProjection{
					ConfigMap: &corev1.ConfigMapProjection{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cf.Name,
						},
					},
				})
			}
		}

		stylingFilesVolume = &corev1.Volume{
			Name: constants.ConfigMapStylingFilesVolumeName,
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: stylingFilesVolumeProjections,
				},
			},
		}
		volumes = append(volumes, figVolume, lgVolume)
	}

	// Add mapfilegenerator config and styling-files (if applicable) here to get the same order as the ansible operator
	// Needed to compare deployments from the ansible operator and this one
	if obj.Mapfile() == nil {
		volumes = append(volumes, corev1.Volume{
			Name:         constants.ConfigMapMapfileGeneratorVolumeName,
			VolumeSource: newVolumeSource(configMapNames.MapfileGenerator),
		})
		if stylingFilesVolume != nil {
			volumes = append(volumes, *stylingFilesVolume)
		}
	}

	return volumes
}

func GetVolumeMountsForDeployment[O pdoknlv3.WMSWFS](obj O) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      constants.BaseVolumeName,
			MountPath: "/srv/data",
		},
		{
			Name:      "data",
			MountPath: "/var/www",
		},
	}

	staticFiles, _ := static.GetStaticFiles()
	for _, name := range staticFiles {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      constants.MapserverName,
			MountPath: "/srv/mapserver/config/" + name,
			SubPath:   name,
		})
	}

	// Custom mapfile
	if mapfile := obj.Mapfile(); mapfile != nil {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      constants.ConfigMapCustomMapfileVolumeName,
			MountPath: "/srv/data/config/mapfile",
		})
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

func GetEnvVarsForDeployment[O pdoknlv3.WMSWFS](obj O, blobsSecretName string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "SERVICE_TYPE",
			Value: string(obj.Type()),
		}, {
			Name:  "MAPSERVER_CONFIG_FILE",
			Value: "/srv/mapserver/config/default_mapserver.conf",
		},
		GetMapfileEnvVar(obj),
		{
			Name: "AZURE_STORAGE_CONNECTION_STRING",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: blobsSecretName, // TODO
					},
					Key: "AZURE_STORAGE_CONNECTION_STRING",
				},
			},
		},
	}
}

// TODO fix linting (cyclop,funlen)
// Resources for mapserver container
//
//nolint:cyclop,funlen
func GetResourcesForDeployment[O pdoknlv3.WMSWFS](obj O) corev1.ResourceRequirements {
	resources := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}

	maxResourceVal := func(v1 *resource.Quantity, v2 *resource.Quantity) *resource.Quantity {
		switch {
		case v1 != nil && v2 != nil:
			if v1.Value() > v2.Value() {
				return v1
			}
			return v2
		case v1 != nil && v2 == nil:
			return v1
		case v1 == nil || v2 != nil:
			return v2
		default:

		}

		return &resource.Quantity{}
	}

	objResources := &corev1.ResourceRequirements{}
	if obj.PodSpecPatch() != nil {
		found := false
		for _, container := range obj.PodSpecPatch().Containers {
			if container.Name == constants.MapserverName {
				objResources = &container.Resources
				found = true
				break
			}
		}

		if !found && obj.PodSpecPatch().Resources != nil {
			objResources = obj.PodSpecPatch().Resources
		}

	}

	/**
	Set CPU request and limit
	*/
	cpuRequest := objResources.Requests.Cpu()
	if cpuRequest == nil || cpuRequest.IsZero() {
		cpuRequest = smoothoperatorutils.Pointer(resource.MustParse("0.15"))
		if obj.Type() == pdoknlv3.ServiceTypeWMS && obj.Options().UseWebserviceProxy() {
			cpuRequest = smoothoperatorutils.Pointer(resource.MustParse("0.1"))
		}
	}
	resources.Requests[corev1.ResourceCPU] = *cpuRequest

	cpuLimit := objResources.Limits.Cpu()
	if cpuLimit != nil && !cpuLimit.IsZero() {
		resources.Limits[corev1.ResourceCPU] = *maxResourceVal(cpuLimit, cpuRequest)
	}

	/**
	Set memory limit/request if the request is higher than the limit the request is used as limit
	*/
	memoryRequest := objResources.Requests.Memory()
	if memoryRequest != nil && !memoryRequest.IsZero() {
		resources.Requests[corev1.ResourceMemory] = *memoryRequest
	}

	memoryLimit := objResources.Limits.Memory()
	if memoryLimit == nil || memoryLimit.IsZero() {
		memoryLimit = smoothoperatorutils.Pointer(resource.MustParse("800M"))
	}
	resources.Limits[corev1.ResourceMemory] = *maxResourceVal(memoryLimit, memoryRequest)

	/**
	Set ephemeral-storage if there is no ephemeral volume
	*/
	// TODO fix linting (nestif)
	if use, _ := mapperutils.UseEphemeralVolume(obj); !use {
		ephemeralStorageRequest := mapperutils.EphemeralStorageRequest(obj)
		if ephemeralStorageRequest != nil {
			resources.Requests[corev1.ResourceEphemeralStorage] = *ephemeralStorageRequest
		}

		ephemeralStorageLimit := mapperutils.EphemeralStorageLimit(obj)
		defaultEphemeralStorageLimit := resource.MustParse("200M")
		if ephemeralStorageLimit == nil || ephemeralStorageLimit.IsZero() || ephemeralStorageLimit.Value() < defaultEphemeralStorageLimit.Value() {
			ephemeralStorageLimit = smoothoperatorutils.Pointer(defaultEphemeralStorageLimit)
		}
		resources.Limits[corev1.ResourceEphemeralStorage] = *maxResourceVal(ephemeralStorageLimit, ephemeralStorageRequest)
	}

	return resources
}

func GetProbesForDeployment[O pdoknlv3.WMSWFS](obj O) (livenessProbe *corev1.Probe, readinessProbe *corev1.Probe, startupProbe *corev1.Probe, err error) {
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
	queryString, err := wfs.ReadinessQueryString()
	if err != nil {
		return nil, err
	}
	return getProbe(queryString, mimeTextXML), nil
}

func getReadinessProbeForWMS(wms *pdoknlv3.WMS) (*corev1.Probe, error) {
	queryString, err := wms.ReadinessQueryString()
	if err != nil {
		return nil, err
	}
	mimeType := "image/png"

	return getProbe(queryString, mimeType), nil
}

func getStartupProbeForWFS(wfs *pdoknlv3.WFS) (*corev1.Probe, error) {
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
