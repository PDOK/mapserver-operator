package controller

import (
	"context"
	"errors"
	"fmt"
	"github.com/pdok/mapserver-operator/internal/controller/featureinfogenerator"
	"github.com/pdok/mapserver-operator/internal/controller/legendgenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	"github.com/pdok/mapserver-operator/internal/controller/ogcwebserviceproxy"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	appsv1 "k8s.io/api/apps/v1"
	"strconv"
	"strings"
	"time"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/blobdownload"
	"github.com/pdok/mapserver-operator/internal/controller/capabilitiesgenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapfilegenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	"github.com/pdok/mapserver-operator/internal/controller/static_files"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	traefikdynamic "github.com/traefik/traefik/v3/pkg/config/dynamic"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	reconciledConditionType          = "Reconciled"
	reconciledConditionReasonSuccess = "Success"
	reconciledConditionReasonError   = "Error"
)

const (
	downloadScriptName         = "gpkg_download.sh"
	mapfileGeneratorInput      = "input.json"
	srvDir                     = "/srv"
	blobsConfigName            = "blobs-config"
	blobsSecretName            = "blobs-secret"
	capabilitiesGeneratorInput = "input.yaml"
	postgisConfigName          = "postgisConfig"
	postgisSecretName          = "postgisSecret"
)

var (
	AppLabelKey   = "app"
	MapserverName = "mapserver"

	// Service ports
	mapserverPortName              = "mapserver"
	mapserverPortNr                = 80
	mapserverWebserviceProxyPortNr = 9111
	metricPortName                 = "metric"
	metricPortNr                   = 9117

	corsHeadersName = "mapserver-headers"
)

type Reconciler interface {
	*WFSReconciler | *WMSReconciler
	client.StatusClient
}

type Images struct {
	MapserverImage             string
	MultitoolImage             string
	MapfileGeneratorImage      string
	CapabilitiesGeneratorImage string
	FeatureinfoGeneratorImage  string
	OgcWebserviceProxyImage    string
}

func getReconcilerClient[R Reconciler](r R) client.Client {
	switch any(r).(type) {
	case *WFSReconciler:
		return any(r).(*WFSReconciler).Client
	case *WMSReconciler:
		return any(r).(*WMSReconciler).Client
	}

	return nil
}

func getReconcilerScheme[R Reconciler](r R) *runtime.Scheme {
	switch any(r).(type) {
	case *WFSReconciler:
		return any(r).(*WFSReconciler).Scheme
	case *WMSReconciler:
		return any(r).(*WMSReconciler).Scheme
	}

	return nil
}

func getReconcilerImages[R Reconciler](r R) *Images {
	switch any(r).(type) {
	case *WFSReconciler:
		return &any(r).(*WFSReconciler).Images
	case *WMSReconciler:
		return &any(r).(*WMSReconciler).Images
	}

	return nil
}

func getSharedBareObjects(obj metav1.Object) []client.Object {
	return []client.Object{
		getBareDeployment(obj, MapserverName),
		getBareIngressRoute(obj),
		getBareHorizontalPodAutoScaler(obj),
		getBareConfigMapBlobDownload(obj),
		getBareConfigMap(obj),
		getBareService(obj),
		getBareCorsHeadersMiddleware(obj),
		getBarePodDisruptionBudget(obj),
		getBareConfigMapMapfileGenerator(obj),
		getBareConfigMapCapabilitiesGenerator(obj),
	}
}

func getBareDeployment(obj metav1.Object, mapserverName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + mapserverName,
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

	matchLabels := smoothoperatorutils.CloneOrEmptyMap(labels)
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: matchLabels,
	}

	deployment.Spec.RevisionHistoryLimit = smoothoperatorutils.Pointer(int32(1))

	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{
				IntVal: 1,
			},
			MaxSurge: &intstr.IntOrString{
				IntVal: 1,
			},
		},
	}

	initContainers, err := getInitContainerForDeployment(r, obj)
	if err != nil {
		return err
	}

	containers, err := getContainersForDeployment(r, obj)
	if err != nil {
		return err
	}

	deployment.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      labels,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: smoothoperatorutils.Pointer(int64(60)),
			InitContainers:                initContainers,
			Volumes:                       mapserver.GetVolumesForDeployment(obj, configMapNames),
			Containers:                    containers,
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, deployment, deployment); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, deployment, getReconcilerScheme(r))

}

func getInitContainerForDeployment[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O) ([]corev1.Container, error) {
	images := getReconcilerImages(r)
	blobDownloadInitContainer, err := blobdownload.GetBlobDownloadInitContainer(obj, images.MultitoolImage, blobsConfigName, blobsSecretName, srvDir)
	if err != nil {
		return nil, err
	}
	mapfileGeneratorInitContainer, err := mapfilegenerator.GetMapfileGeneratorInitContainer(obj, images.MapfileGeneratorImage, postgisConfigName, postgisSecretName, srvDir)
	if err != nil {
		return nil, err
	}
	capabilitiesGeneratorInitContainer, err := capabilitiesgenerator.GetCapabilitiesGeneratorInitContainer(obj, images.CapabilitiesGeneratorImage)
	if err != nil {
		return nil, err
	}

	initContainers := []corev1.Container{
		*blobDownloadInitContainer,
		*mapfileGeneratorInitContainer,
		*capabilitiesGeneratorInitContainer,
	}

	if wms, ok := any(obj).(*pdoknlv3.WMS); ok {
		legendGeneratorInitContainer, err := legendgenerator.GetLegendGeneratorInitContainer(wms, images.MapserverImage, srvDir)
		if err != nil {
			return nil, err
		}
		initContainers = append(initContainers, *legendGeneratorInitContainer)

		featureInfoInitContainer, err := featureinfogenerator.GetFeatureinfoGeneratorInitContainer(images.FeatureinfoGeneratorImage, srvDir)
		if err != nil {
			return nil, err
		}
		initContainers = append(initContainers, *featureInfoInitContainer)

		if *wms.Options().RewriteGroupToDataLayers {
			legendFixerInitContainer := legendgenerator.GetLegendFixerInitContainer(images.MultitoolImage)
			initContainers = append(initContainers, *legendFixerInitContainer)
		}

	}
	return initContainers, nil
}

func getContainersForDeployment[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O) ([]corev1.Container, error) {
	images := getReconcilerImages(r)

	livenessProbe, readinessProbe, startupProbe, err := mapserver.GetProbesForDeployment(obj)
	if err != nil {
		return nil, err
	}

	containers := []corev1.Container{
		{
			Name:            MapserverName,
			Image:           images.MapserverImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 80,
				},
			},
			Env:            mapserver.GetEnvVarsForDeployment(obj, blobsSecretName),
			VolumeMounts:   mapserver.GetVolumeMountsForDeployment(obj, srvDir),
			Resources:      mapserver.GetResourcesForDeployment(obj),
			LivenessProbe:  livenessProbe,
			ReadinessProbe: readinessProbe,
			StartupProbe:   startupProbe,
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					Sleep: &corev1.SleepAction{Seconds: 15},
				},
			},
		},
	}

	if wms, ok := any(obj).(*pdoknlv3.WMS); ok {
		if wms.Options().UseWebserviceProxy() {
			ogcWebserviceProxyContainer, err := ogcwebserviceproxy.GetOgcWebserviceProxyContainer(wms, images.OgcWebserviceProxyImage)
			if err != nil {
				return nil, err
			}
			containers = append(containers, *ogcWebserviceProxyContainer)
		}
	}

	return containers, nil
}

func getBareIngressRoute(obj metav1.Object) *traefikiov1alpha1.IngressRoute {
	return &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateIngressRoute[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, ingressRoute *traefikiov1alpha1.IngressRoute) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, ingressRoute, labels); err != nil {
		return err
	}

	var uptimeURL string
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		uptimeURL = any(obj).(*pdoknlv3.WFS).Spec.Service.URL // TODO add healthcheck query
	case *pdoknlv3.WMS:
		uptimeURL = any(obj).(*pdoknlv3.WMS).Spec.Service.URL // TODO add healthcheck query
	}

	uptimeName, err := makeUptimeName(obj)
	if err != nil {
		return err
	}
	annotations := smoothoperatorutils.CloneOrEmptyMap(obj.GetAnnotations())
	annotations["uptime.pdok.nl/id"] = obj.ID()
	annotations["uptime.pdok.nl/name"] = uptimeName
	annotations["uptime.pdok.nl/url"] = uptimeURL
	annotations["uptime.pdok.nl/tags"] = strings.Join(makeUptimeTags(obj), ",")
	ingressRoute.SetAnnotations(annotations)

	mapserverService := traefikiov1alpha1.Service{
		LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
			Name: getBareService(obj).GetName(),
			Kind: "Service",
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: int32(mapserverPortNr),
			},
		},
	}

	middlewareRef := traefikiov1alpha1.MiddlewareRef{
		Name:      getBareCorsHeadersMiddleware(obj).GetName(),
		Namespace: obj.GetNamespace(),
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		wms, _ := any(obj).(*pdoknlv3.WMS)
		ingressRoute.Spec.Routes = []traefikiov1alpha1.Route{{
			Kind:        "Rule",
			Match:       getLegendMatchRule(wms),
			Services:    []traefikiov1alpha1.Service{mapserverService},
			Middlewares: []traefikiov1alpha1.MiddlewareRef{middlewareRef},
		}}

		if obj.Options().UseWebserviceProxy() {
			webServiceProxyService := traefikiov1alpha1.Service{
				LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
					Name: getBareService(obj).GetName(),
					Kind: "Service",
					Port: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(mapserverWebserviceProxyPortNr),
					},
				},
			}

			ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, traefikiov1alpha1.Route{
				Kind:        "Rule",
				Match:       getMatchRule(obj),
				Services:    []traefikiov1alpha1.Service{webServiceProxyService},
				Middlewares: []traefikiov1alpha1.MiddlewareRef{middlewareRef},
			})
		} else {
			ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, traefikiov1alpha1.Route{
				Kind:        "Rule",
				Match:       getMatchRule(obj),
				Services:    []traefikiov1alpha1.Service{mapserverService},
				Middlewares: []traefikiov1alpha1.MiddlewareRef{middlewareRef},
			})
		}
	} else { // WFS
		ingressRoute.Spec.Routes = []traefikiov1alpha1.Route{{
			Kind:        "Rule",
			Match:       getMatchRule(obj),
			Services:    []traefikiov1alpha1.Service{mapserverService},
			Middlewares: []traefikiov1alpha1.MiddlewareRef{middlewareRef},
		}}
	}

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, ingressRoute, ingressRoute); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, ingressRoute, getReconcilerScheme(r))
}

func makeUptimeTags[O pdoknlv3.WMSWFS](obj O) []string {
	tags := []string{"public-stats", strings.ToLower(string(obj.Type()))}

	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		wfs, _ := any(obj).(*pdoknlv3.WFS)
		if wfs.Spec.Service.Inspire != nil {
			tags = append(tags, "inspire")
		}
	case *pdoknlv3.WMS:
		wms, _ := any(obj).(*pdoknlv3.WMS)
		if wms.Spec.Service.Inspire != nil {
			tags = append(tags, "inspire")
		}
	}

	return tags
}

func makeUptimeName[O pdoknlv3.WMSWFS](obj O) (string, error) {
	var parts []string

	inspire := false
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		inspire = any(obj).(*pdoknlv3.WFS).Spec.Service.Inspire != nil
	case *pdoknlv3.WMS:
		inspire = any(obj).(*pdoknlv3.WMS).Spec.Service.Inspire != nil
	}

	ownerID, ok := obj.GetLabels()["dataset-owner"]
	if !ok {
		return "", errors.New("dataset-owner label not found in object")
	}
	parts = append(parts, strings.ToUpper(strings.ReplaceAll(ownerID, "-", "")))

	datasetID, ok := obj.GetLabels()["dataset"]
	if !ok {
		return "", errors.New("dataset label not found in object")
	}
	parts = append(parts, strings.ReplaceAll(datasetID, "-", ""))

	theme, ok := obj.GetLabels()["theme"]
	if ok {
		parts = append(parts, strings.ReplaceAll(theme, "-", ""))
	}

	version, ok := obj.GetLabels()["service-version"]
	if !ok {
		return "", errors.New("service-version label not found in object")
	}
	parts = append(parts, version)

	if inspire {
		parts = append(parts, "INSPIRE")
	}

	parts = append(parts, string(obj.Type()))

	return strings.Join(parts, " "), nil
}

func getMatchRule[O pdoknlv3.WMSWFS](obj O) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && Path(`/" + pdoknlv3.GetBaseURLPath(obj) + "`)"
}

func getLegendMatchRule(wms *pdoknlv3.WMS) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && Path(`/" + pdoknlv3.GetBaseURLPath(wms) + "/legend`)"
}

func getBareConfigMapMapfileGenerator(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-mapfile-generator",
			Namespace: obj.GetNamespace(),
		},
	}
}

func getBareConfigMapCapabilitiesGenerator(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-capabilities-generator",
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateConfigMapCapabilitiesGenerator[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		input, err := capabilitiesgenerator.GetInput(obj, ownerInfo)
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{capabilitiesGeneratorInput: input}
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(obj, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func mutateConfigMapMapfileGenerator[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		//mapfileGeneratorConfig, err := mapfilegenerator.GetConfig(obj, ownerInfo)
		//		if err != nil {
		//			return err
		//		}
		mapfileGeneratorConfig := "TODO" // TODO Implement mapfilegenerator.GetConfig for WMS
		configMap.Data = map[string]string{mapfileGeneratorInput: mapfileGeneratorConfig}
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(obj, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func getBareHorizontalPodAutoScaler(obj metav1.Object) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateHorizontalPodAutoscaler[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, autoscaler *autoscalingv2.HorizontalPodAutoscaler) error {
	autoscalerPatch := obj.HorizontalPodAutoscalerPatch()
	podSpecPatch := obj.PodSpecPatch()
	var behaviourStabilizationWindowSeconds int32 = 0
	if obj.Type() == pdoknlv3.ServiceTypeWFS {
		behaviourStabilizationWindowSeconds = 300
	}

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(getReconcilerClient(r), autoscaler, labels); err != nil {
		return err
	}

	minReplicas := int32(2)
	if autoscalerPatch != nil && autoscalerPatch.MinReplicas != nil {
		minReplicas = *autoscalerPatch.MinReplicas
	}

	maxReplicas := int32(30)
	if autoscalerPatch != nil && autoscalerPatch.MaxReplicas != 0 {
		maxReplicas = autoscalerPatch.MaxReplicas
	}

	var metrics []autoscalingv2.MetricSpec
	if autoscalerPatch != nil {
		metrics = autoscalerPatch.Metrics
	}
	if len(metrics) == 0 {
		var avgU int32 = 90
		if podSpecPatch != nil && podSpecPatch.Resources.Requests.Cpu() != nil {
			avgU = 80
		}
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: smoothoperatorutils.Pointer(avgU),
				},
			},
		})
	}

	autoscaler.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
			Kind: "Deployment",
			Name: obj.GetName() + "-" + MapserverName,
		},
		MinReplicas: &minReplicas,
		MaxReplicas: maxReplicas,
		Metrics:     metrics,
		Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
			ScaleUp: &autoscalingv2.HPAScalingRules{
				StabilizationWindowSeconds: &behaviourStabilizationWindowSeconds,
				SelectPolicy:               smoothoperatorutils.Pointer(autoscalingv2.MaxChangePolicySelect),
				Policies: []autoscalingv2.HPAScalingPolicy{
					{
						Type:          autoscalingv2.PodsScalingPolicy,
						Value:         20,
						PeriodSeconds: 60,
					},
				},
			},
			ScaleDown: &autoscalingv2.HPAScalingRules{
				StabilizationWindowSeconds: smoothoperatorutils.Pointer(int32(3600)),
				SelectPolicy:               smoothoperatorutils.Pointer(autoscalingv2.MaxChangePolicySelect),
				Policies: []autoscalingv2.HPAScalingPolicy{
					{
						Type:          autoscalingv2.PodsScalingPolicy,
						Value:         1,
						PeriodSeconds: 600,
					},
					{
						Type:          autoscalingv2.PercentScalingPolicy,
						Value:         10,
						PeriodSeconds: 600,
					},
				},
			},
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(getReconcilerClient(r), autoscaler, autoscaler); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, autoscaler, getReconcilerScheme(r))
}

func getBareConfigMapBlobDownload(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-init-scripts",
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateConfigMapBlobDownload[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		downloadScript := blobdownload.GetScript()
		configMap.Data = map[string]string{downloadScriptName: downloadScript}
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(obj, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func getBareService(obj metav1.Object) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateService[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, service *corev1.Service) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	selector := smoothoperatorutils.CloneOrEmptyMap(labels)
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, service, labels); err != nil {
		return err
	}

	ports := []corev1.ServicePort{
		{
			Name:     mapserverPortName,
			Port:     int32(mapserverPortNr),
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     metricPortName,
			Port:     int32(metricPortNr),
			Protocol: corev1.ProtocolTCP,
		},
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		if obj.Options().UseWebserviceProxy() {
			ports = append(ports, corev1.ServicePort{
				Name: "ogc-webservice-proxy",
				Port: 9111,
			})
		}
	}

	service.Spec = corev1.ServiceSpec{
		Ports:    ports,
		Selector: selector,
	}
	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, service, service); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, service, getReconcilerScheme(r))
}

func getBareCorsHeadersMiddleware(obj metav1.Object) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + corsHeadersName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
			UID:       obj.GetUID(),
		},
	}
}

func mutateCorsHeadersMiddleware[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, middleware *traefikiov1alpha1.Middleware) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		Headers: &traefikdynamic.Headers{
			CustomResponseHeaders: map[string]string{
				"Access-Control-Allow-Headers": "Content-Type",
				"Access-Control-Allow-Method":  "GET, HEAD, OPTIONS",
				"Access-Control-Allow-Origin":  "*",
				"Cache-Control":                "public, max-age=3600, no-transform",
			},
		},
	}
	// TODO - do we need this in WFS/WMS
	// middleware.Spec.Headers.FrameDeny = true

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, middleware, middleware); err != nil {
		return err
	}

	return ctrl.SetControllerReference(obj, middleware, getReconcilerScheme(r))
}

func getBarePodDisruptionBudget(obj metav1.Object) *v1.PodDisruptionBudget {
	return &v1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutatePodDisruptionBudget[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, podDisruptionBudget *v1.PodDisruptionBudget) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, podDisruptionBudget, labels); err != nil {
		return err
	}

	matchLabels := smoothoperatorutils.CloneOrEmptyMap(labels)
	podDisruptionBudget.Spec = v1.PodDisruptionBudgetSpec{
		MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Selector: &metav1.LabelSelector{
			MatchLabels: matchLabels,
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, podDisruptionBudget, podDisruptionBudget); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, podDisruptionBudget, getReconcilerScheme(r))
}

func getBareConfigMap(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateConfigMap[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, configMap, labels); err != nil {
		return err
	}

	configMap.Immutable = smoothoperatorutils.Pointer(true)
	configMap.Data = map[string]string{}

	for name, content := range static_files.GetStaticFiles() {
		if name == "include.conf" {
			content = []byte(strings.ReplaceAll(string(content), "/{{ service_path }}", mapperutils.GetPath(obj)))
		}
		configMap.Data[name] = string(content)
	}

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(obj, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func addCommonLabels[O pdoknlv3.WMSWFS](obj O, labels map[string]string) map[string]string {
	labels[AppLabelKey] = MapserverName

	inspire := false
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		inspire = any(obj).(*pdoknlv3.WFS).Spec.Service.Inspire != nil
	case *pdoknlv3.WMS:
		inspire = any(obj).(*pdoknlv3.WMS).Spec.Service.Inspire != nil
	}

	labels["inspire"] = strconv.FormatBool(inspire)

	return labels
}

func logAndUpdateStatusError[R Reconciler](ctx context.Context, r R, obj client.Object, err error) {
	var generation int64

	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		generation = any(obj).(*pdoknlv3.WFS).Generation
	case *pdoknlv3.WMS:
		generation = any(obj).(*pdoknlv3.WMS).Generation
	}

	updateStatus(ctx, r, obj, []metav1.Condition{{
		Type:               reconciledConditionType,
		Status:             metav1.ConditionFalse,
		Reason:             reconciledConditionReasonError,
		Message:            err.Error(),
		ObservedGeneration: generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}}, nil)
}

func logAndUpdateStatusFinished[R Reconciler](ctx context.Context, r R, obj client.Object, operationResults map[string]controllerutil.OperationResult) {
	lgr := log.FromContext(ctx)
	lgr.Info("operation results", "results", operationResults)

	var generation int64

	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		generation = any(obj).(*pdoknlv3.WFS).Generation
	case *pdoknlv3.WMS:
		generation = any(obj).(*pdoknlv3.WMS).Generation
	}

	updateStatus(ctx, r, obj, []metav1.Condition{{
		Type:               reconciledConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             reconciledConditionReasonSuccess,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}}, operationResults)
}

func updateStatus[R Reconciler](ctx context.Context, r R, obj client.Object, conditions []metav1.Condition, operationResults map[string]controllerutil.OperationResult) {
	lgr := log.FromContext(ctx)
	if err := getReconcilerClient(r).Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		log.FromContext(ctx).Error(err, "unable to update status")
		return
	}

	var status *model.OperatorStatus
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		status = &any(obj).(*pdoknlv3.WFS).Status
	case *pdoknlv3.WMS:
		status = &any(obj).(*pdoknlv3.WMS).Status
	}

	changed := false
	for _, condition := range conditions {
		if meta.SetStatusCondition(&status.Conditions, condition) {
			changed = true
		}
	}
	if !equality.Semantic.DeepEqual(status.OperationResults, operationResults) {
		status.OperationResults = operationResults
		changed = true
	}
	if !changed {
		return
	}
	if err := r.Status().Update(ctx, obj); err != nil {
		lgr.Error(err, "unable to update status")
	}
}

func getFinalizerName[O pdoknlv3.WMSWFS](obj O) string {
	return strings.ToLower(string(obj.Type())) + "." + pdoknlv3.GroupVersion.Group + "/finalizer"
}

func createOrUpdateAllForWMSWFS[R Reconciler, O pdoknlv3.WMSWFS](ctx context.Context, r R, obj O, ownerInfo *smoothoperatorv1.OwnerInfo) (operationResults map[string]controllerutil.OperationResult, err error) {
	operationResults = make(map[string]controllerutil.OperationResult)
	reconcilerClient := getReconcilerClient(r)

	hashedConfigMapNames := types.HashedConfigMapNames{}

	// region ConfigMap
	{
		configMap := getBareConfigMap(obj)
		if err = mutateConfigMap(r, obj, configMap); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, configMap)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, configMap, func() error {
			return mutateConfigMap(r, obj, configMap)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, configMap), err)
		}
		hashedConfigMapNames.ConfigMap = configMap.Name
	}
	// end region ConfigMap

	// region ConfigMap-MapfileGenerator
	{
		configMapMfg := getBareConfigMapMapfileGenerator(obj)
		if err = mutateConfigMapMapfileGenerator(r, obj, configMapMfg, ownerInfo); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapMfg)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, configMapMfg, func() error {
			return mutateConfigMapMapfileGenerator(r, obj, configMapMfg, ownerInfo)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapMfg), err)
		}
		hashedConfigMapNames.MapfileGenerator = configMapMfg.Name
	}
	// end region ConfigMap-MapfileGenerator

	// region ConfigMap-CapabilitiesGenerator
	{
		configMapCg := getBareConfigMapCapabilitiesGenerator(obj)
		if err = mutateConfigMapCapabilitiesGenerator(r, obj, configMapCg, ownerInfo); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapCg)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, configMapCg, func() error {
			return mutateConfigMapCapabilitiesGenerator(r, obj, configMapCg, ownerInfo)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapCg), err)
		}
		hashedConfigMapNames.CapabilitiesGenerator = configMapCg.Name
	}
	// end region ConfigMap-CapabilitiesGenerator

	// region ConfigMap-BlobDownload
	{
		configMapBd := getBareConfigMapBlobDownload(obj)
		if err = mutateConfigMapBlobDownload(r, obj, configMapBd); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapBd)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, configMapBd, func() error {
			return mutateConfigMapBlobDownload(r, obj, configMapBd)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapBd), err)
		}
		hashedConfigMapNames.BlobDownload = configMapBd.Name
	}
	// end region ConfigMap-BlobDownload

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		wms, _ := any(obj).(*pdoknlv3.WMS)
		wmsReconciler := (*WMSReconciler)(r)

		// region ConfigMap-LegendGenerator
		{
			configMapLg := getBareConfigMapLegendGenerator(obj)
			if err = mutateConfigMapLegendGenerator(wmsReconciler, wms, configMapLg); err != nil {
				return operationResults, err
			}
			operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapLg)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, configMapLg, func() error {
				return mutateConfigMapLegendGenerator(wmsReconciler, wms, configMapLg)
			})
			if err != nil {
				return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapLg), err)
			}
			hashedConfigMapNames.LegendGenerator = configMapLg.Name
		}
		// end region ConfigMap-LegendGenerator

		// region ConfigMap-FeatureinfoGenerator
		{
			configMapFig := getBareConfigMapFeatureinfoGenerator(obj)
			if err = mutateConfigMapFeatureinfoGenerator(wmsReconciler, wms, configMapFig); err != nil {
				return operationResults, err
			}
			operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapFig)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, configMapFig, func() error {
				return mutateConfigMapFeatureinfoGenerator(wmsReconciler, wms, configMapFig)
			})
			if err != nil {
				return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapFig), err)
			}
			hashedConfigMapNames.FeatureInfoGenerator = configMapFig.Name
		}

		// end region ConfigMap-FeatureinfoGenerator

		// region ConfigMap-OgcWebserviceProxy
		{
			configMapOwp := getBareConfigMapOgcWebserviceProxy(obj)
			if err = mutateConfigMapOgcWebserviceProxy(wmsReconciler, wms, configMapOwp); err != nil {
				return operationResults, err
			}
			operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapOwp)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, configMapOwp, func() error {
				return mutateConfigMapOgcWebserviceProxy(wmsReconciler, wms, configMapOwp)
			})
			if err != nil {
				return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, configMapOwp), err)
			}
			hashedConfigMapNames.OgcWebserviceProxy = configMapOwp.Name
		}
		// end  region ConfigMap-OgcWebserviceProxy
	}

	// region Deployment
	{
		deployment := getBareDeployment(obj, MapserverName)
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, deployment)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, deployment, func() error {
			return mutateDeployment(r, obj, deployment, hashedConfigMapNames)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, deployment), err)
		}
	}
	// end region Deployment

	// region TraefikMiddleware
	{
		middleware := getBareCorsHeadersMiddleware(obj)
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, middleware)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, middleware, func() error {
			return mutateCorsHeadersMiddleware(r, obj, middleware)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, middleware), err)
		}
	}
	// end region TraefikMiddleware

	// region PodDisruptionBudget
	{
		podDisruptionBudget := getBarePodDisruptionBudget(obj)
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, podDisruptionBudget)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, podDisruptionBudget, func() error {
			return mutatePodDisruptionBudget(r, obj, podDisruptionBudget)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, podDisruptionBudget), err)
		}
	}
	// end region PodDisruptionBudget

	// region HorizontalAutoScaler
	{
		autoscaler := getBareHorizontalPodAutoScaler(obj)
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, autoscaler)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, autoscaler, func() error {
			return mutateHorizontalPodAutoscaler(r, obj, autoscaler)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, autoscaler), err)
		}
	}
	// end region HorizontalAutoScaler

	// region IngressRoute
	{
		ingress := getBareIngressRoute(obj)
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, ingress)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, ingress, func() error {
			return mutateIngressRoute(r, obj, ingress)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, ingress), err)
		}
	}
	// end region IngressRoute

	// region Service
	{
		service := getBareService(obj)
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, service)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, service, func() error {
			return mutateService(r, obj, service)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, service), err)
		}
	}
	// end region Service

	return operationResults, nil
}

func deleteAllForWMSWFS[R Reconciler, O pdoknlv3.WMSWFS](ctx context.Context, r R, obj O, ownerInfo *smoothoperatorv1.OwnerInfo) (err error) {
	bareObjects := getSharedBareObjects(obj)
	var objects []client.Object

	// Remove ConfigMaps as they have hashed names
	for _, object := range bareObjects {
		if _, ok := object.(*corev1.ConfigMap); !ok {
			objects = append(objects, object)
		}
	}

	// ConfigMap
	cm := getBareConfigMap(obj)
	err = mutateConfigMap(r, obj, cm)
	if err != nil {
		return err
	}
	objects = append(objects, cm)

	// ConfigMap-MapfileGenerator
	cmMg := getBareConfigMapMapfileGenerator(obj)
	err = mutateConfigMapMapfileGenerator(r, obj, cmMg, ownerInfo)
	if err != nil {
		return err
	}
	objects = append(objects, cmMg)

	// ConfigMap-CapabilitiesGenerator
	cmCg := getBareConfigMapCapabilitiesGenerator(obj)
	err = mutateConfigMapCapabilitiesGenerator(r, obj, cmCg, ownerInfo)
	if err != nil {
		return err
	}
	objects = append(objects, cmCg)

	// ConfigMap-BlobDownload
	cmBd := getBareConfigMapBlobDownload(obj)
	err = mutateConfigMapBlobDownload(r, obj, cmBd)
	if err != nil {
		return err
	}
	objects = append(objects, cmBd)

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		wms, _ := any(obj).(*pdoknlv3.WMS)
		wmsReconciler := (*WMSReconciler)(r)

		// ConfigMap-LegendGenerator
		cmLg := getBareConfigMapLegendGenerator(obj)
		err = mutateConfigMapLegendGenerator(wmsReconciler, wms, cmLg)
		if err != nil {
			return err
		}
		objects = append(objects, cmLg)

		// ConfigMap-FeatureInfo
		cmFi := getBareConfigMapFeatureinfoGenerator(obj)
		err = mutateConfigMapFeatureinfoGenerator(wmsReconciler, wms, cmFi)
		if err != nil {
			return err
		}
		objects = append(objects, cmFi)

		// ConfigMap-OgcWebserviceProxy
		cmOwp := getBareConfigMapOgcWebserviceProxy(obj)
		err = mutateConfigMapOgcWebserviceProxy(wmsReconciler, wms, cmOwp)
		if err != nil {
			return err
		}
		objects = append(objects, cmOwp)
	}

	return smoothoperatorutils.DeleteObjects(ctx, getReconcilerClient(r), objects)
}
