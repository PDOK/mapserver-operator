package mapserver

import (
	"os"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	"github.com/pdok/mapserver-operator/internal/controller/static_files"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConfigMapVolumeName                      = "mapserver"
	ConfigMapMapfileGeneratorVolumeName      = "mapfile-generator-config"
	ConfigMapCapabilitiesGeneratorVolumeName = "capabilities-generator-config"
	ConfigMapBlobDownloadVolumeName          = "init-scripts"
)

func GetBareDeployment(obj metav1.Object, mapserverName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + mapserverName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

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

	volumes := []v1.Volume{
		baseVolume,
		{
			Name: "data",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: ConfigMapVolumeName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: configMapNames.ConfigMap,
					},
				},
			},
		},
		{
			Name: ConfigMapMapfileGeneratorVolumeName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: configMapNames.MapfileGenerator},
				},
			},
		},
		{
			Name: ConfigMapCapabilitiesGeneratorVolumeName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: configMapNames.CapabilitiesGenerator},
				},
			},
		},
	}

	if mapfile := obj.Mapfile(); mapfile != nil {
		volumes = append(volumes, v1.Volume{
			Name: "mapfile",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: mapfile.ConfigMapKeyRef.Name,
					},
				},
			},
		})
	}

	if options := obj.Options(); options != nil {
		if options.PrefetchData != nil && *options.PrefetchData {
			volumes = append(volumes, v1.Volume{
				Name: ConfigMapBlobDownloadVolumeName,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{Name: configMapNames.BlobDownload},
						DefaultMode:          smoothoperatorutils.Pointer(int32(0777)),
					},
				},
			})
		}
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

	for name := range static_files.GetStaticFiles() {
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

func GetEnvVarsForDeployment[O pdoknlv3.WMSWFS](obj O, blobsSecretName string) []v1.EnvVar {
	mapFileName := "service.map"
	if obj.Mapfile() != nil {
		mapFileName = obj.Mapfile().ConfigMapKeyRef.Key
	}

	return []v1.EnvVar{
		{
			Name:  "SERVICE_TYPE",
			Value: string(obj.Type()),
		}, {
			Name:  "MAPSERVER_CONFIG_FILE",
			Value: "/srv/mapserver/config/default_mapserver.conf",
		}, {
			Name:  "MS_MAPFILE",
			Value: mapFileName,
		}, {
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
	}
}

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
	if obj.PodSpecPatch() != nil && obj.PodSpecPatch().Resources != nil {
		objResources = obj.PodSpecPatch().Resources
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		if options := obj.Options(); options != nil {
			if options.DisableWebserviceProxy == nil || !*options.DisableWebserviceProxy {
				resources.Requests[v1.ResourceCPU] = resource.MustParse("0.1")
			}
		}
	}

	if objResources.Limits.Cpu() != nil && objResources.Requests.Cpu().Value() > resources.Requests.Cpu().Value() {
		resources.Limits[v1.ResourceCPU] = *objResources.Limits.Cpu()
	}

	if objResources.Requests.Memory() != nil {
		resources.Requests[v1.ResourceMemory] = *objResources.Requests.Memory()
	}

	if use, _ := mapperutils.UseEphemeralVolume(obj); !use {
		value := mapperutils.EphemeralStorageLimit(obj)

		if value.Value() > minimumEphemeralStorageLimit.Value() {
			resources.Limits[v1.ResourceEphemeralStorage] = *value
		}
	}

	ephemeralStorageRequest := mapperutils.EphemeralStorageRequest(obj)
	if ephemeralStorageRequest != nil {
		resources.Requests[v1.ResourceEphemeralStorage] = *ephemeralStorageRequest
	}

	return resources
}
