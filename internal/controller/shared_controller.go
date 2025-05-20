package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"

	"k8s.io/apimachinery/pkg/api/resource"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/blobdownload"
	"github.com/pdok/mapserver-operator/internal/controller/capabilitiesgenerator"
	"github.com/pdok/mapserver-operator/internal/controller/featureinfogenerator"
	"github.com/pdok/mapserver-operator/internal/controller/legendgenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapfilegenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	"github.com/pdok/mapserver-operator/internal/controller/ogcwebserviceproxy"
	"github.com/pdok/mapserver-operator/internal/controller/static"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"github.com/pdok/smooth-operator/model"
	smoothoperatork8s "github.com/pdok/smooth-operator/pkg/k8s"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	traefikdynamic "github.com/traefik/traefik/v3/pkg/config/dynamic"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
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
	blobsConfigPrefix          = "blobs-"
	blobsSecretPrefix          = "blobs-"
	capabilitiesGeneratorInput = "input.yaml"
	postgisConfigPrefix        = "postgres-"
	postgisSecretPrefix        = "postgres-"
)

var (
	AppLabelKey               = "app"
	MapserverName             = "mapserver"
	LegendGeneratorName       = "legend-generator"
	FeatureInfoGeneratorName  = "featureinfo-generator"
	OgcWebserviceProxyName    = "ogc-webservice-proxy"
	MapfileGeneratorName      = "mapfile-generator"
	CapabilitiesGeneratorName = "capabilities-generator"
	InitScriptsName           = "init-scripts"

	// Service ports
	mapserverPortName                    = "mapserver"
	mapserverPortNr                int32 = 80
	mapserverWebserviceProxyPortNr       = 9111
	metricPortName                       = "metric"
	metricPortNr                   int32 = 9117

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
	ApacheExporterImage        string
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

func getSharedBareObjects[O pdoknlv3.WMSWFS](obj O) []client.Object {
	return []client.Object{
		getBareDeployment(obj),
		getBareIngressRoute(obj),
		getBareHorizontalPodAutoScaler(obj),
		getBareConfigMap(obj, InitScriptsName),
		getBareConfigMap(obj, MapserverName),
		getBareService(obj),
		getBareCorsHeadersMiddleware(obj),
		getBarePodDisruptionBudget(obj),
		getBareConfigMap(obj, MapfileGeneratorName),
		getBareConfigMap(obj, CapabilitiesGeneratorName),
	}
}

func getSuffixedName[O pdoknlv3.WMSWFS](obj O, suffix string) string {
	return obj.GetName() + "-" + strings.ToLower(string(obj.Type())) + "-" + suffix
}

func getBareDeployment[O pdoknlv3.WMSWFS](obj O) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: getSuffixedName(obj, MapserverName),
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
	for idx := range initContainers {
		initContainers[idx].TerminationMessagePolicy = corev1.TerminationMessagePolicy("File")
		initContainers[idx].TerminationMessagePath = "/dev/termination-log"
	}

	containers, err := getContainersForDeployment(r, obj)
	if err != nil {
		return err
	}

	annotations := smoothoperatorutils.CloneOrEmptyMap(deployment.Spec.Template.GetAnnotations())
	annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"] = "true"

	annotations["kubectl.kubernetes.io/default-container"] = "mapserver"
	annotations["match-regex.version-checker.io/mapserver"] = `^\d\.\d\.\d.*$`
	annotations["prometheus.io/scrape"] = "true"
	annotations["prometheus.io/port"] = "9117"
	annotations["priority.version-checker.io/mapserver"] = "4"
	annotations["priority.version-checker.io/ogc-webservice-proxy"] = "4"

	deployment.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                 corev1.RestartPolicyAlways,
			TerminationGracePeriodSeconds: smoothoperatorutils.Pointer(int64(60)),
			InitContainers:                initContainers,
			Volumes:                       mapserver.GetVolumesForDeployment(obj, configMapNames),
			Containers:                    containers,
			SecurityContext:               deployment.Spec.Template.Spec.SecurityContext,
			SchedulerName:                 deployment.Spec.Template.Spec.SchedulerName,
			DNSPolicy:                     corev1.DNSClusterFirst,
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, deployment, deployment); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, deployment, getReconcilerScheme(r))
}

func getInitContainerForDeployment[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O) ([]corev1.Container, error) {
	blobsConfig, err := smoothoperatork8s.GetConfigMap(getReconcilerClient(r), obj.GetNamespace(), blobsConfigPrefix, make(map[string]string))
	if err != nil {
		return nil, err
	}

	blobsSecret, err := smoothoperatork8s.GetSecret(getReconcilerClient(r), obj.GetNamespace(), blobsSecretPrefix, make(map[string]string))
	if err != nil {
		return nil, err
	}

	postgresConfig, err := smoothoperatork8s.GetConfigMap(getReconcilerClient(r), obj.GetNamespace(), postgisConfigPrefix, make(map[string]string))
	if err != nil {
		return nil, err
	}

	postgresSecret, err := smoothoperatork8s.GetSecret(getReconcilerClient(r), obj.GetNamespace(), postgisSecretPrefix, make(map[string]string))
	if err != nil {
		return nil, err
	}

	images := getReconcilerImages(r)
	blobDownloadInitContainer, err := blobdownload.GetBlobDownloadInitContainer(obj, images.MultitoolImage, blobsConfig.Name, blobsSecret.Name, srvDir)
	if err != nil {
		return nil, err
	}
	mapfileGeneratorInitContainer, err := mapfilegenerator.GetMapfileGeneratorInitContainer(obj, images.MapfileGeneratorImage, postgresConfig.Name, postgresSecret.Name, srvDir)
	if err != nil {
		return nil, err
	}
	capabilitiesGeneratorInitContainer, err := capabilitiesgenerator.GetCapabilitiesGeneratorInitContainer(obj, images.CapabilitiesGeneratorImage)
	if err != nil {
		return nil, err
	}

	initContainers := []corev1.Container{
		*blobDownloadInitContainer,
		*capabilitiesGeneratorInitContainer,
	}

	if obj.Mapfile() == nil {
		initContainers = append(initContainers, *mapfileGeneratorInitContainer)
	}

	if wms, ok := any(obj).(*pdoknlv3.WMS); ok {
		featureInfoInitContainer, err := featureinfogenerator.GetFeatureinfoGeneratorInitContainer(images.FeatureinfoGeneratorImage, srvDir)
		if err != nil {
			return nil, err
		}
		initContainers = append(initContainers, *featureInfoInitContainer)

		legendGeneratorInitContainer, err := legendgenerator.GetLegendGeneratorInitContainer(wms, images.MapserverImage, srvDir)
		if err != nil {
			return nil, err
		}
		initContainers = append(initContainers, *legendGeneratorInitContainer)

		if wms.Options().RewriteGroupToDataLayers {
			legendFixerInitContainer := legendgenerator.GetLegendFixerInitContainer(images.MultitoolImage)
			initContainers = append(initContainers, *legendFixerInitContainer)
		}

	}
	return initContainers, nil
}

func getContainersForDeployment[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O) ([]corev1.Container, error) {
	images := getReconcilerImages(r)

	blobsSecret, err := smoothoperatork8s.GetSecret(getReconcilerClient(r), obj.GetNamespace(), blobsSecretPrefix, make(map[string]string))
	if err != nil {
		return nil, err
	}

	livenessProbe, readinessProbe, startupProbe, err := mapserver.GetProbesForDeployment(obj)
	if err != nil {
		return nil, err
	}

	mapserverContainer := corev1.Container{
		Name:            MapserverName,
		Image:           images.MapserverImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 80,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env:                      mapserver.GetEnvVarsForDeployment(obj, blobsSecret.Name),
		VolumeMounts:             mapserver.GetVolumeMountsForDeployment(obj, srvDir),
		Resources:                mapserver.GetResourcesForDeployment(obj),
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		TerminationMessagePath:   "/dev/termination-log",
		LivenessProbe:            livenessProbe,
		ReadinessProbe:           readinessProbe,
		StartupProbe:             startupProbe,
		Lifecycle: &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"sleep", "15"},
				},
			},
		},
	}

	apacheContainer := corev1.Container{
		Name:                     "apache-exporter",
		Image:                    images.ApacheExporterImage,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		TerminationMessagePath:   "/dev/termination-log",
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 9117,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Args: []string{
			"--scrape_uri=http://localhost/server-status?auto",
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("48M"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("0.02"),
			},
		},
	}

	containers := []corev1.Container{
		mapserverContainer,
		apacheContainer,
	}
	if wms, ok := any(obj).(*pdoknlv3.WMS); ok {
		if wms.Options().UseWebserviceProxy() {
			ogcWebserviceProxyContainer, err := ogcwebserviceproxy.GetOgcWebserviceProxyContainer(wms, images.OgcWebserviceProxyImage)
			if err != nil {
				return nil, err
			}

			return append(containers, *ogcWebserviceProxyContainer), nil
		}
	}

	return containers, nil
}

func getBareIngressRoute[O pdoknlv3.WMSWFS](obj O) *traefikiov1alpha1.IngressRoute {
	return &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSuffixedName(obj, MapserverName),
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
				IntVal: mapserverPortNr,
			},
		},
	}

	middlewareRef := traefikiov1alpha1.MiddlewareRef{
		Name: getBareCorsHeadersMiddleware(obj).GetName(),
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
						Type: intstr.Int,
						//nolint:gosec
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

	// Add finalizers
	ingressRoute.Finalizers = []string{"uptime.pdok.nl/finalizer"}

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
	host := pdoknlv3.GetHost(false)
	if strings.Contains(host, "localhost") {
		return "Host(`localhost`) && Path(`/" + pdoknlv3.GetBaseURLPath(obj) + "`)"
	}

	return "(Host(`localhost`) || Host(`" + host + "`)) && Path(`/" + pdoknlv3.GetBaseURLPath(obj) + "`)"
}

func getLegendMatchRule(wms *pdoknlv3.WMS) string {
	host := pdoknlv3.GetHost(false)
	if strings.Contains(host, "localhost") {
		return "Host(`localhost`) && PathPrefix(`/" + pdoknlv3.GetBaseURLPath(wms) + "/legend`)"
	}

	return "(Host(`localhost`) || Host(`" + host + "`)) && PathPrefix(`/" + pdoknlv3.GetBaseURLPath(wms) + "/legend`)"
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
		mapfileGeneratorConfig, err := mapfilegenerator.GetConfig(obj, ownerInfo)
		if err != nil {
			return err
		}
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

func getBareHorizontalPodAutoScaler[O pdoknlv3.WMSWFS](obj O) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSuffixedName(obj, MapserverName),
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateHorizontalPodAutoscaler[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, autoscaler *autoscalingv2.HorizontalPodAutoscaler) error {
	autoscalerPatch := obj.HorizontalPodAutoscalerPatch()
	var behaviourStabilizationWindowSeconds int32
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
		if cpu := mapperutils.GetContainerResourceRequest(obj, "mapserver", corev1.ResourceCPU); cpu != nil {
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
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       getSuffixedName(obj, MapserverName),
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
						Type:          autoscalingv2.PercentScalingPolicy,
						Value:         10,
						PeriodSeconds: 600,
					},
					{
						Type:          autoscalingv2.PodsScalingPolicy,
						Value:         1,
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

func getBareService[O pdoknlv3.WMSWFS](obj O) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSuffixedName(obj, MapserverName),
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
			Name:       mapserverPortName,
			Port:       mapserverPortNr,
			TargetPort: intstr.FromInt32(mapserverPortNr),
			Protocol:   corev1.ProtocolTCP,
		},
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		if obj.Options().UseWebserviceProxy() {
			ports = append(ports, corev1.ServicePort{
				Name: OgcWebserviceProxyName,
				Port: 9111,
			})
		}
	}

	// Add port here to get the same port order as the odl ansible operator
	ports = append(ports, corev1.ServicePort{
		Name:       metricPortName,
		Port:       metricPortNr,
		TargetPort: intstr.FromInt32(metricPortNr),
		Protocol:   corev1.ProtocolTCP,
	})

	service.Spec = corev1.ServiceSpec{
		Type:                  corev1.ServiceTypeClusterIP,
		ClusterIP:             service.Spec.ClusterIP,
		ClusterIPs:            service.Spec.ClusterIPs,
		IPFamilyPolicy:        service.Spec.IPFamilyPolicy,
		IPFamilies:            service.Spec.IPFamilies,
		SessionAffinity:       corev1.ServiceAffinityNone,
		InternalTrafficPolicy: smoothoperatorutils.Pointer(corev1.ServiceInternalTrafficPolicyCluster),
		Ports:                 ports,
		Selector:              selector,
	}
	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, service, service); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, service, getReconcilerScheme(r))
}

func getBareCorsHeadersMiddleware[O pdoknlv3.WMSWFS](obj O) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: getSuffixedName(obj, corsHeadersName),
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
				"Access-Control-Allow-Method":  "GET, POST, OPTIONS",
				"Access-Control-Allow-Origin":  "*",
				"Cache-Control":                "public, max-age=3600, no-transform",
			},
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, middleware, middleware); err != nil {
		return err
	}

	return ctrl.SetControllerReference(obj, middleware, getReconcilerScheme(r))
}

func getBarePodDisruptionBudget[O pdoknlv3.WMSWFS](obj O) *v1.PodDisruptionBudget {
	return &v1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSuffixedName(obj, MapserverName),
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

func getBareConfigMap[O pdoknlv3.WMSWFS](obj O, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSuffixedName(obj, name),
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

	staticFileName, contents := static.GetStaticFiles()
	for _, name := range staticFileName {
		content := contents[name]
		if name == "include.conf" {
			content = []byte(strings.ReplaceAll(string(content), "{{ service_path }}", pdoknlv3.GetBaseURLPath(obj)))
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

func createOrUpdateAllForWMSWFS[R Reconciler, O pdoknlv3.WMSWFS](ctx context.Context, r R, obj O, ownerInfo *smoothoperatorv1.OwnerInfo) (operationResults map[string]controllerutil.OperationResult, err error) {
	reconcilerClient := getReconcilerClient(r)

	hashedConfigMapNames, operationResults, err := createOrUpdateConfigMaps(ctx, r, obj, ownerInfo)
	if err != nil {
		return operationResults, err
	}

	// region Deployment
	{
		deployment := getBareDeployment(obj)
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, deployment)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, deployment, func() error {
			return mutateDeployment(r, obj, deployment, hashedConfigMapNames)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, deployment), err)
		}
	}
	// end region Deployment

	// region TraefikMiddleware
	if obj.Options().IncludeIngress {
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
	if obj.Options().IncludeIngress {
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

func createOrUpdateConfigMaps[R Reconciler, O pdoknlv3.WMSWFS](ctx context.Context, r R, obj O, ownerInfo *smoothoperatorv1.OwnerInfo) (hashedConfigMapNames types.HashedConfigMapNames, operationResults map[string]controllerutil.OperationResult, err error) {
	operationResults, configMaps := make(map[string]controllerutil.OperationResult), make(map[string]func(R, O, *corev1.ConfigMap) error)
	configMaps[MapserverName] = mutateConfigMap
	if obj.Mapfile() == nil {
		configMaps[MapfileGeneratorName] = func(r R, o O, cm *corev1.ConfigMap) error {
			return mutateConfigMapMapfileGenerator(r, o, cm, ownerInfo)
		}
	}
	configMaps[CapabilitiesGeneratorName] = func(r R, o O, cm *corev1.ConfigMap) error {
		return mutateConfigMapCapabilitiesGenerator(r, o, cm, ownerInfo)
	}
	if obj.Options().PrefetchData {
		configMaps[InitScriptsName] = mutateConfigMapBlobDownload
	}
	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		wms, _ := any(obj).(*pdoknlv3.WMS)
		wmsReconciler := (*WMSReconciler)(r)

		configMaps[LegendGeneratorName] = func(_ R, _ O, cm *corev1.ConfigMap) error {
			return mutateConfigMapLegendGenerator(wmsReconciler, wms, cm)
		}
		configMaps[FeatureInfoGeneratorName] = func(_ R, _ O, cm *corev1.ConfigMap) error {
			return mutateConfigMapFeatureinfoGenerator(wmsReconciler, wms, cm)
		}
		configMaps[OgcWebserviceProxyName] = func(_ R, _ O, cm *corev1.ConfigMap) error {
			return mutateConfigMapOgcWebserviceProxy(wmsReconciler, wms, cm)
		}
	}
	for cmName, mutate := range configMaps {
		cm, or, err := createOrUpdateConfigMap(ctx, obj, r, cmName, func(r R, o O, cm *corev1.ConfigMap) error {
			return mutate(r, o, cm)
		})
		if or != nil {
			operationResults[smoothoperatorutils.GetObjectFullName(getReconcilerClient(r), cm)] = *or
		}
		if err != nil {
			return hashedConfigMapNames, operationResults, err
		}
		switch cmName {
		case MapserverName:
			hashedConfigMapNames.ConfigMap = cm.Name
		case MapfileGeneratorName:
			hashedConfigMapNames.MapfileGenerator = cm.Name
		case CapabilitiesGeneratorName:
			hashedConfigMapNames.CapabilitiesGenerator = cm.Name
		case InitScriptsName:
			hashedConfigMapNames.BlobDownload = cm.Name
		case LegendGeneratorName:
			hashedConfigMapNames.LegendGenerator = cm.Name
		case FeatureInfoGeneratorName:
			hashedConfigMapNames.FeatureInfoGenerator = cm.Name
		case OgcWebserviceProxyName:
			hashedConfigMapNames.OgcWebserviceProxy = cm.Name
		}
	}

	return hashedConfigMapNames, operationResults, err
}

func createOrUpdateConfigMap[O pdoknlv3.WMSWFS, R Reconciler](ctx context.Context, obj O, reconciler R, name string, mutate func(R, O, *corev1.ConfigMap) error) (*corev1.ConfigMap, *controllerutil.OperationResult, error) {
	reconcilerClient := getReconcilerClient(reconciler)
	cm := getBareConfigMap(obj, name)
	if err := mutate(reconciler, obj, cm); err != nil {
		return cm, nil, err
	}
	or, err := controllerutil.CreateOrUpdate(ctx, reconcilerClient, cm, func() error {
		return mutate(reconciler, obj, cm)
	})
	if err != nil {
		return cm, &or, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, cm), err)
	}
	return cm, &or, nil
}
