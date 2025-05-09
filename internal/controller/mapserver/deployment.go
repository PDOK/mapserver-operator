package mapserver

import (
	"errors"
	"os"
	"strings"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	"github.com/pdok/mapserver-operator/internal/controller/static_files"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	ConfigMapVolumeName                      = "mapserver"
	ConfigMapMapfileGeneratorVolumeName      = "mapfile-generator-config"
	ConfigMapCapabilitiesGeneratorVolumeName = "capabilities-generator-config"
	ConfigMapBlobDownloadVolumeName          = "init-scripts"
	ConfigMapOgcWebserviceProxyVolumeName    = "ogc-webservice-proxy-config"
	ConfigMapLegendGeneratorVolumeName       = "legend-generator-config"
	ConfigMapFeatureinfoGeneratorVolumeName  = "featureinfo-generator-config"
	ConfigMapStylingFilesVolumeName          = "styling-files"
	// TODO How should we determine this boundingbox?
	healthCheckBbox = "190061.4619730016857,462435.5987861062749,202917.7508707302331,473761.6884966178914"

	mimeTextXML = "text/xml"
)

func GetVolumesForDeployment[O pdoknlv3.WMSWFS](obj O, configMapNames types.HashedConfigMapNames) []v1.Volume {
	baseVolume := v1.Volume{Name: "base"}
	if use, size := mapperutils.UseEphemeralVolume(obj); use {
		baseVolume.Ephemeral = &v1.EphemeralVolumeSource{
			VolumeClaimTemplate: &v1.PersistentVolumeClaimTemplate{
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
					Resources: v1.VolumeResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceStorage: *size,
						},
					},
				},
			},
		}

		if value, set := os.LookupEnv("STORAGE_CLASS_NAME"); set {
			baseVolume.Ephemeral.VolumeClaimTemplate.Spec.StorageClassName = &value
		}
	} else {
		baseVolume.VolumeSource = v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		}
	}

	newVolumeSource := func(name string) v1.VolumeSource {
		return v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				DefaultMode: smoothoperatorutils.Pointer(int32(0644)),
				LocalObjectReference: v1.LocalObjectReference{
					Name: name,
				},
			},
		}
	}

	volumes := []v1.Volume{
		baseVolume,
		{
			Name: "data",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
		{
			Name:         ConfigMapVolumeName,
			VolumeSource: newVolumeSource(configMapNames.ConfigMap),
		},
		{
			Name:         ConfigMapMapfileGeneratorVolumeName,
			VolumeSource: newVolumeSource(configMapNames.MapfileGenerator),
		},
		{
			Name:         ConfigMapCapabilitiesGeneratorVolumeName,
			VolumeSource: newVolumeSource(configMapNames.CapabilitiesGenerator),
		},
	}

	if mapfile := obj.Mapfile(); mapfile != nil {
		volumes = append(volumes, v1.Volume{
			Name:         "mapfile",
			VolumeSource: newVolumeSource(mapfile.ConfigMapKeyRef.Key),
		})
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		lgVolume := v1.Volume{
			Name:         ConfigMapLegendGeneratorVolumeName,
			VolumeSource: newVolumeSource(configMapNames.LegendGenerator),
		}
		figVolume := v1.Volume{
			Name:         ConfigMapFeatureinfoGeneratorVolumeName,
			VolumeSource: newVolumeSource(configMapNames.FeatureInfoGenerator),
		}

		wms, _ := any(obj).(*pdoknlv3.WMS)
		stylingFilesVolumeProjections := []v1.VolumeProjection{}
		if wms.Spec.Service.StylingAssets != nil && wms.Spec.Service.StylingAssets.ConfigMapRefs != nil {
			for _, cf := range wms.Spec.Service.StylingAssets.ConfigMapRefs {
				stylingFilesVolumeProjections = append(stylingFilesVolumeProjections, v1.VolumeProjection{
					ConfigMap: &v1.ConfigMapProjection{
						LocalObjectReference: v1.LocalObjectReference{
							Name: cf.Name,
						},
					},
				})
			}
		}

		stylingFilesVolume := v1.Volume{
			Name: ConfigMapStylingFilesVolumeName,
			VolumeSource: v1.VolumeSource{
				Projected: &v1.ProjectedVolumeSource{
					Sources: stylingFilesVolumeProjections,
				},
			},
		}
		volumes = append(volumes, lgVolume, figVolume, stylingFilesVolume)
	}

	if obj.Options().PrefetchData {
		vol := newVolumeSource(configMapNames.BlobDownload)
		vol.ConfigMap.DefaultMode = smoothoperatorutils.Pointer(int32(0777))
		volumes = append(volumes, v1.Volume{
			Name:         ConfigMapBlobDownloadVolumeName,
			VolumeSource: vol,
		})
	}
	if obj.Type() == pdoknlv3.ServiceTypeWMS && obj.Options().UseWebserviceProxy() {
		volumes = append(volumes, v1.Volume{
			Name:         ConfigMapOgcWebserviceProxyVolumeName,
			VolumeSource: newVolumeSource(configMapNames.OgcWebserviceProxy),
		})
	}

	return volumes
}

func GetVolumeMountsForDeployment[O pdoknlv3.WMSWFS](obj O, srvDir string) []v1.VolumeMount {
	volumeMounts := []v1.VolumeMount{
		{
			Name:      "base",
			MountPath: "/srv/data",
		},
		{
			Name:      "data",
			MountPath: "/var/www",
		},
	}

	staticFiles, _ := static_files.GetStaticFiles()
	for _, name := range staticFiles {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "mapserver",
			MountPath: srvDir + "/mapserver/config/" + name,
			SubPath:   name,
		})
	}

	// Custom mapfile
	if mapfile := obj.Mapfile(); mapfile != nil {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "mapfile",
			MountPath: "/srv/data/config/mapfile",
		})
	}

	return volumeMounts
}

func GetMapfileEnvVar[O pdoknlv3.WMSWFS](obj O) v1.EnvVar {
	mapFileName := "service.map"
	if obj.Mapfile() != nil {
		mapFileName = obj.Mapfile().ConfigMapKeyRef.Key
	}

	return v1.EnvVar{
		Name:  "MS_MAPFILE",
		Value: "/srv/data/config/mapfile/" + mapFileName,
	}
}

func GetEnvVarsForDeployment[O pdoknlv3.WMSWFS](obj O, blobsSecretName string) []v1.EnvVar {
	return []v1.EnvVar{
		{
			Name:  "SERVICE_TYPE",
			Value: string(obj.Type()),
		}, {
			Name:  "MAPSERVER_CONFIG_FILE",
			Value: "/srv/mapserver/config/default_mapserver.conf",
		},
		{
			Name: "AZURE_STORAGE_CONNECTION_STRING",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: blobsSecretName, // TODO
					},
					Key: "AZURE_STORAGE_CONNECTION_STRING",
				},
			},
		},
		GetMapfileEnvVar(obj),
	}
}

// Resources for mapserver container
func GetResourcesForDeployment[O pdoknlv3.WMSWFS](obj O) v1.ResourceRequirements {
	minimumEphemeralStorageLimit := resource.MustParse("200M")
	resources := v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceMemory:           resource.MustParse("800M"),
			v1.ResourceEphemeralStorage: minimumEphemeralStorageLimit,
		},
		Requests: v1.ResourceList{
			v1.ResourceCPU: resource.MustParse("0.15"),
		},
	}

	objResources := &v1.ResourceRequirements{}
	if obj.PodSpecPatch() != nil {
		found := false
		for _, container := range obj.PodSpecPatch().Containers {
			if container.Name == "mapserver" {
				objResources = &container.Resources
				found = true
				break
			}
		}

		if !found && obj.PodSpecPatch().Resources != nil {
			objResources = obj.PodSpecPatch().Resources
		}

	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS && obj.Options().UseWebserviceProxy() {
		resources.Requests[v1.ResourceCPU] = resource.MustParse("0.1")
	}

	if objResources.Limits.Cpu() != nil && objResources.Requests.Cpu().Value() > resources.Requests.Cpu().Value() {
		resources.Limits[v1.ResourceCPU] = *objResources.Limits.Cpu()
	}

	if objResources.Requests.Memory() != nil && !objResources.Requests.Memory().IsZero() {
		resources.Requests[v1.ResourceMemory] = *objResources.Requests.Memory()
	}

	if use, _ := mapperutils.UseEphemeralVolume(obj); !use {
		value := mapperutils.EphemeralStorageLimit(obj)

		if objResources.Limits.StorageEphemeral() != nil && objResources.Limits.StorageEphemeral().Value() > minimumEphemeralStorageLimit.Value() {
			resources.Limits[v1.ResourceEphemeralStorage] = *value
		}
	}

	ephemeralStorageRequest := mapperutils.EphemeralStorageRequest(obj)
	if ephemeralStorageRequest != nil {
		resources.Requests[v1.ResourceEphemeralStorage] = *ephemeralStorageRequest
	}

	return resources
}

func GetProbesForDeployment[O pdoknlv3.WMSWFS](obj O) (livenessProbe *v1.Probe, readinessProbe *v1.Probe, startupProbe *v1.Probe, err error) {
	livenessProbe = getLivenessProbe(obj)
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		wfs, _ := any(obj).(*pdoknlv3.WFS)
		readinessProbe, err = getReadinessProbeForWFS(wfs)
		if err != nil {
			return nil, nil, nil, err
		}
		startupProbe, err = getStartupProbeForWFS(wfs)
		if err != nil {
			return nil, nil, nil, err
		}
	case *pdoknlv3.WMS:
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

func getLivenessProbe[O pdoknlv3.WMSWFS](obj O) *v1.Probe {
	webserviceType := strings.ToLower(string(obj.Type()))
	queryString := "SERVICE=" + webserviceType + "&request=GetCapabilities"
	return getProbe(queryString, mimeTextXML)
}

func getReadinessProbeForWFS(wfs *pdoknlv3.WFS) (*v1.Probe, error) {
	if len(wfs.Spec.Service.FeatureTypes) == 0 {
		return nil, errors.New("cannot get readiness probe for WFS, featuretypes could not be found")
	}
	queryString := "SERVICE=WFS&VERSION=2.0.0&REQUEST=GetFeature&TYPENAMES=" + wfs.Spec.Service.FeatureTypes[0].Name + "&STARTINDEX=0&COUNT=1"
	return getProbe(queryString, mimeTextXML), nil
}

func getReadinessProbeForWMS(wms *pdoknlv3.WMS) (*v1.Probe, error) {
	firstDataLayerName := ""
	for _, layer := range wms.Spec.Service.Layer.GetAllLayers() {
		if layer.IsDataLayer() {
			firstDataLayerName = *layer.Name
			break
		}
	}
	if firstDataLayerName == "" {
		return nil, errors.New("cannot get readiness probe for WMS, the first datalayer could not be found")
	}

	queryString := "SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&BBOX=" + healthCheckBbox + "&CRS=EPSG:28992&WIDTH=100&HEIGHT=100&LAYERS=" + firstDataLayerName + "&STYLES=&FORMAT=image/png"
	mimeType := "image/png"

	return getProbe(queryString, mimeType), nil
}

func getStartupProbeForWFS(wfs *pdoknlv3.WFS) (*v1.Probe, error) {
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

func getStartupProbeForWMS(wms *pdoknlv3.WMS) (*v1.Probe, error) {
	var layerNames []string
	for _, layer := range wms.Spec.Service.Layer.GetAllLayers() {
		if layer.Name != nil {
			layerNames = append(layerNames, *layer.Name)
		}

	}
	if len(layerNames) == 0 {
		return nil, errors.New("cannot get startup probe for WMS, layers could not be found")
	}

	queryString := "SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&BBOX=" + healthCheckBbox + "&CRS=EPSG:28992&WIDTH=100&HEIGHT=100&LAYERS=" + strings.Join(layerNames, ",") + "&STYLES=&FORMAT=image/png"
	mimeType := "image/png"
	return getProbe(queryString, mimeType), nil
}

func getProbe(queryString string, mimeType string) *v1.Probe {
	probeCmd := "wget -SO- -T 10 -t 2 'http://127.0.0.1:80/mapserver?" + queryString + "' 2>&1 | egrep -aiA10 'HTTP/1.1 200' | egrep -i 'Content-Type: " + mimeType + "'"
	return &v1.Probe{
		ProbeHandler: v1.ProbeHandler{Exec: &v1.ExecAction{
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
