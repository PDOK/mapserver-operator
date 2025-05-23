package controller

import (
	"strconv"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/blobdownload"
	"github.com/pdok/mapserver-operator/internal/controller/capabilitiesgenerator"
	"github.com/pdok/mapserver-operator/internal/controller/constants"
	"github.com/pdok/mapserver-operator/internal/controller/featureinfogenerator"
	"github.com/pdok/mapserver-operator/internal/controller/legendgenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapfilegenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	"github.com/pdok/mapserver-operator/internal/controller/ogcwebserviceproxy"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	"github.com/pdok/smooth-operator/pkg/k8s"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	blobsConfigPrefix   = "blobs-"
	blobsSecretPrefix   = "blobs-"
	postgisConfigPrefix = "postgres-"
	postgisSecretPrefix = "postgres-"
)

var storageClassName string

func SetStorageClassName(name string) {
	storageClassName = name
}

func getBareDeployment[O pdoknlv3.WMSWFS](obj O) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: getSuffixedName(obj, constants.MapserverName),
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateDeployment[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, deployment *appsv1.Deployment, configMapNames types.HashedConfigMapNames) error {
	reconcilerClient := getReconcilerClient(r)
	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, deployment, labels); err != nil {
		return err
	}

	deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	deployment.Spec.RevisionHistoryLimit = smoothoperatorutils.Pointer(int32(1))
	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{IntVal: 1},
			MaxSurge:       &intstr.IntOrString{IntVal: 1},
		},
	}

	blobsSecret, err := k8s.GetSecret(getReconcilerClient(r), obj.GetNamespace(), blobsSecretPrefix, make(map[string]string))
	if err != nil {
		return err
	}

	initContainers, err := getInitContainerForDeployment(r, obj)
	if err != nil {
		return err
	}
	setTerminationMessage(initContainers)

	images := getReconcilerImages(r)
	containers, err := getContainers(obj, images, blobsSecret)
	if err != nil {
		return err
	}
	setTerminationMessage(containers)

	volumes := getVolumes(obj, configMapNames)

	podTemplateSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: getPodAnnotations(deployment),
			Labels:      labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                 corev1.RestartPolicyAlways,
			DNSPolicy:                     corev1.DNSClusterFirst,
			TerminationGracePeriodSeconds: smoothoperatorutils.Pointer(int64(60)),
			InitContainers:                initContainers,
			Containers:                    containers,
			Volumes:                       volumes,
		},
	}

	if obj.PodSpecPatch() != nil {
		patchedSpec, err := smoothoperatorutils.StrategicMergePatch(&podTemplateSpec.Spec, obj.PodSpecPatch())
		if err != nil {
			return err
		}
		podTemplateSpec.Spec = *patchedSpec
	}

	if use, _ := mapperutils.UseEphemeralVolume(obj); !use {
		ephStorage := podTemplateSpec.Spec.Containers[0].Resources.Limits[corev1.ResourceEphemeralStorage]
		threshold := resource.MustParse("200M")

		if ephStorage.Value() < threshold.Value() {
			podTemplateSpec.Spec.Containers[0].Resources.Limits[corev1.ResourceEphemeralStorage] = threshold
		}
	} else {
		delete(podTemplateSpec.Spec.Containers[0].Resources.Limits, corev1.ResourceEphemeralStorage)
		delete(podTemplateSpec.Spec.Containers[0].Resources.Requests, corev1.ResourceEphemeralStorage)
	}

	deployment.Spec.Template = podTemplateSpec

	if err = smoothoperatorutils.EnsureSetGVK(reconcilerClient, deployment, deployment); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, deployment, getReconcilerScheme(r))
}

func getPodAnnotations(deployment *appsv1.Deployment) map[string]string {
	annotations := smoothoperatorutils.CloneOrEmptyMap(deployment.Spec.Template.GetAnnotations())
	annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"] = "true"
	annotations["kubectl.kubernetes.io/default-container"] = constants.MapserverName
	annotations["match-regex.version-checker.io/mapserver"] = `^\d\.\d\.\d.*$`
	annotations["prometheus.io/scrape"] = "true"
	annotations["prometheus.io/port"] = strconv.Itoa(int(constants.ApachePortNr))
	annotations["priority.version-checker.io/mapserver"] = "4"
	annotations["priority.version-checker.io/ogc-webservice-proxy"] = "4"
	return annotations
}

func getInitContainerForDeployment[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O) ([]corev1.Container, error) {
	blobsConfig, err := k8s.GetConfigMap(getReconcilerClient(r), obj.GetNamespace(), blobsConfigPrefix, make(map[string]string))
	if err != nil {
		return nil, err
	}

	blobsSecret, err := k8s.GetSecret(getReconcilerClient(r), obj.GetNamespace(), blobsSecretPrefix, make(map[string]string))
	if err != nil {
		return nil, err
	}

	images := getReconcilerImages(r)
	blobDownloadInitContainer, err := blobdownload.GetBlobDownloadInitContainer(obj, *images, blobsConfig.Name, blobsSecret.Name)
	if err != nil {
		return nil, err
	}
	capabilitiesGeneratorInitContainer, err := capabilitiesgenerator.GetCapabilitiesGeneratorInitContainer(obj, *images)
	if err != nil {
		return nil, err
	}

	initContainers := []corev1.Container{
		*blobDownloadInitContainer,
		*capabilitiesGeneratorInitContainer,
	}

	if obj.Mapfile() == nil {
		postgresConfig, err := k8s.GetConfigMap(getReconcilerClient(r), obj.GetNamespace(), postgisConfigPrefix, make(map[string]string))
		if err != nil {
			return nil, err
		}

		postgresSecret, err := k8s.GetSecret(getReconcilerClient(r), obj.GetNamespace(), postgisSecretPrefix, make(map[string]string))
		if err != nil {
			return nil, err
		}
		mapfileGeneratorInitContainer, err := mapfilegenerator.GetMapfileGeneratorInitContainer(obj, *images, postgresConfig.Name, postgresSecret.Name)
		if err != nil {
			return nil, err
		}
		initContainers = append(initContainers, *mapfileGeneratorInitContainer)
	}

	if wms, ok := any(obj).(*pdoknlv3.WMS); ok {
		featureInfoInitContainer, err := featureinfogenerator.GetFeatureinfoGeneratorInitContainer(*images)
		if err != nil {
			return nil, err
		}
		initContainers = append(initContainers, *featureInfoInitContainer)

		legendGeneratorInitContainer, err := legendgenerator.GetLegendGeneratorInitContainer(wms, *images)
		if err != nil {
			return nil, err
		}
		initContainers = append(initContainers, *legendGeneratorInitContainer)

		if wms.Options().RewriteGroupToDataLayers {
			legendFixerInitContainer := legendgenerator.GetLegendFixerInitContainer(*images)
			initContainers = append(initContainers, *legendFixerInitContainer)
		}

	}
	return initContainers, nil
}

func getContainers[O pdoknlv3.WMSWFS](obj O, images *types.Images, blobsSecret *corev1.Secret) ([]corev1.Container, error) {
	mapserverContainer, err := mapserver.GetMapserverContainer(obj, *images, blobsSecret.Name)
	if err != nil {
		return nil, err
	}
	containers := []corev1.Container{
		*mapserverContainer,
		getApacheContainer(*images),
	}
	if wms, ok := any(obj).(*pdoknlv3.WMS); ok {
		ogcProxy, err := ogcwebserviceproxy.GetOgcWebserviceProxyContainer(wms, *images)
		if err != nil {
			return nil, err
		}
		containers = append(containers, *ogcProxy)
	}
	return containers, nil
}

func getApacheContainer(images types.Images) corev1.Container {
	return corev1.Container{
		Name:            "apache-exporter",
		Image:           images.ApacheExporterImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports:           []corev1.ContainerPort{{ContainerPort: constants.ApachePortNr, Protocol: corev1.ProtocolTCP}},
		Args:            []string{"-scrape_uri=http://localhost/server-status?auto"},
		Resources: corev1.ResourceRequirements{
			Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("48M")},
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("0.02")},
		},
	}
}

func setTerminationMessage(c []corev1.Container) {
	for i := range c {
		c[i].TerminationMessagePolicy = "File"
		c[i].TerminationMessagePath = "/dev/termination-log"
	}
}

func getVolumes[O pdoknlv3.WMSWFS](obj O, configMapNames types.HashedConfigMapNames) []corev1.Volume {
	baseVolume := corev1.Volume{Name: constants.BaseVolumeName}
	if use, size := mapperutils.UseEphemeralVolume(obj); use {
		baseVolume.Ephemeral = &corev1.EphemeralVolumeSource{
			VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceStorage: *size,
					}},
				},
			},
		}
		if storageClassName != "" {
			baseVolume.Ephemeral.VolumeClaimTemplate.Spec.StorageClassName = &storageClassName
		}
	} else {
		baseVolume.EmptyDir = &corev1.EmptyDirVolumeSource{}
	}

	volumes := []corev1.Volume{
		baseVolume,
		{Name: constants.DataVolumeName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		getConfigMapVolume(constants.MapserverName, configMapNames.Mapserver),
	}

	if mapfile := obj.Mapfile(); mapfile != nil {
		volumes = append(volumes, getConfigMapVolume(constants.ConfigMapCustomMapfileVolumeName, mapfile.ConfigMapKeyRef.Name))
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS && obj.Options().UseWebserviceProxy() {
		volumes = append(volumes, getConfigMapVolume(constants.ConfigMapOgcWebserviceProxyVolumeName, configMapNames.OgcWebserviceProxy))
	}

	if obj.Options().PrefetchData {
		vol := getConfigMapVolume(constants.InitScriptsName, configMapNames.InitScripts)
		vol.ConfigMap.DefaultMode = smoothoperatorutils.Pointer(int32(0777))
		volumes = append(volumes, vol)
	}

	volumes = append(volumes, getConfigMapVolume(constants.ConfigMapCapabilitiesGeneratorVolumeName, configMapNames.CapabilitiesGenerator))

	if obj.Mapfile() == nil {
		volumes = append(volumes, getConfigMapVolume(constants.ConfigMapMapfileGeneratorVolumeName, configMapNames.MapfileGenerator))
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		if obj.Mapfile() == nil {
			wms, _ := any(obj).(*pdoknlv3.WMS)
			volumeProjections := []corev1.VolumeProjection{}
			for _, cm := range wms.Spec.Service.StylingAssets.ConfigMapRefs {
				volumeProjections = append(volumeProjections, corev1.VolumeProjection{
					ConfigMap: &corev1.ConfigMapProjection{LocalObjectReference: corev1.LocalObjectReference{Name: cm.Name}},
				})
			}

			volumes = append(volumes, corev1.Volume{
				Name:         constants.ConfigMapStylingFilesVolumeName,
				VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: volumeProjections}},
			})
		}

		volumes = append(
			volumes,
			getConfigMapVolume(constants.ConfigMapFeatureinfoGeneratorVolumeName, configMapNames.FeatureInfoGenerator),
			getConfigMapVolume(constants.ConfigMapLegendGeneratorVolumeName, configMapNames.LegendGenerator),
		)
	}

	return volumes
}

func getConfigMapVolume(name, configMap string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			DefaultMode:          smoothoperatorutils.Pointer(int32(0644)),
			LocalObjectReference: corev1.LocalObjectReference{Name: configMap}},
		},
	}
}
