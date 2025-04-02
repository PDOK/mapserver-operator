package controller

import (
	"context"
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/blobdownload"
	"github.com/pdok/mapserver-operator/internal/controller/mapfilegenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
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
	"strings"
	"time"
)

const (
	reconciledConditionType          = "Reconciled"
	reconciledConditionReasonSuccess = "Success"
	reconciledConditionReasonError   = "Error"
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

func GetReconcilerClient[R Reconciler](r R) client.Client {
	switch any(r).(type) {
	case *WFSReconciler:
		return any(r).(*WFSReconciler).Client
	case *WMSReconciler:
		return any(r).(*WMSReconciler).Client
	}

	return nil
}

func GetReconcilerScheme[R Reconciler](r R) *runtime.Scheme {
	switch any(r).(type) {
	case *WFSReconciler:
		return any(r).(*WFSReconciler).Scheme
	case *WMSReconciler:
		return any(r).(*WMSReconciler).Scheme
	}

	return nil
}

func GetSharedBareObjects(obj metav1.Object) []client.Object {
	return []client.Object{
		mapserver.GetBareDeployment(obj, MapserverName),
		GetBareIngressRoute(obj),
		GetBareHorizontalPodAutoScaler(obj),
		GetBareConfigMapBlobDownload(obj),
		GetBareConfigMap(obj),
		GetBareService(obj),
		GetBareCorsHeadersMiddleware(obj),
		GetBarePodDisruptionBudget(obj),
		GetBareConfigMapMapfileGenerator(obj),
		GetBareConfigMapCapabilitiesGenerator(obj),
	}
}

func GetBareIngressRoute(obj metav1.Object) *traefikiov1alpha1.IngressRoute {
	return &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func MutateIngressRoute[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, ingressRoute *traefikiov1alpha1.IngressRoute) error {
	reconcilerClient := GetReconcilerClient(r)

	labels := smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels())
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, ingressRoute, labels); err != nil {
		return err
	}

	uptimeTags := []string{"public-stats", strings.ToLower(string(obj.Type()))}
	var uptimeUrl string
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		wfs, _ := any(obj).(*pdoknlv3.WFS)
		if wfs.Spec.Service.Inspire != nil {
			uptimeTags = append(uptimeTags, "inspire")
		}
		uptimeUrl = wfs.Spec.Service.URL // TODO add healthcheck query
	case *pdoknlv3.WMS:
		wms, _ := any(obj).(*pdoknlv3.WMS)
		if wms.Spec.Service.Inspire != nil {
			uptimeTags = append(uptimeTags, "inspire")
		}
		uptimeUrl = wms.Spec.Service.URL // TODO add healthcheck query
	}
	annotations := smoothoperatorutils.CloneOrEmptyMap(obj.GetAnnotations())
	annotations["uptime.pdok.nl/id"] = obj.Id()
	annotations["uptime.pdok.nl/name"] = obj.GetName() // TODO make uptime name
	annotations["uptime.pdok.nl/url"] = uptimeUrl
	annotations["uptime.pdok.nl/tags"] = strings.Join(uptimeTags, ",")
	ingressRoute.SetAnnotations(annotations)

	mapserverService := traefikiov1alpha1.Service{
		LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
			Name: GetBareService(obj).GetName(),
			Kind: "Service",
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: int32(mapserverPortNr),
			},
		},
	}

	webServiceProxyService := traefikiov1alpha1.Service{
		LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
			Name: GetBareService(obj).GetName(),
			Kind: "Service",
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: int32(mapserverWebserviceProxyPortNr),
			},
		},
	}

	middlewareRef := traefikiov1alpha1.MiddlewareRef{
		Name:      GetBareCorsHeadersMiddleware(obj).GetName(),
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

		if obj.Options() != nil && (obj.Options().DisableWebserviceProxy == nil || *obj.Options().DisableWebserviceProxy == false) {
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
	return ctrl.SetControllerReference(obj, ingressRoute, GetReconcilerScheme(r))
}

func getMatchRule[O pdoknlv3.WMSWFS](obj O) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && Path(`/" + pdoknlv3.GetBaseURLPath(obj) + "`)"
}

func getLegendMatchRule(wms *pdoknlv3.WMS) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && Path(`/" + pdoknlv3.GetBaseURLPath(wms) + "/legend`)"
}

func GetBareConfigMapMapfileGenerator(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-mapfile-generator",
			Namespace: obj.GetNamespace(),
		},
	}
}

func GetBareConfigMapCapabilitiesGenerator(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-capabilities-generator",
			Namespace: obj.GetNamespace(),
		},
	}
}

func MutateConfigMapMapfileGenerator[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	reconcilerClient := GetReconcilerClient(r)

	labels := smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels())
	labels[AppLabelKey] = MapserverName
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
	if err := ctrl.SetControllerReference(obj, configMap, GetReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func GetBareHorizontalPodAutoScaler(obj metav1.Object) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func MutateHorizontalPodAutoscaler[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, autoscaler *autoscalingv2.HorizontalPodAutoscaler) error {
	autoscalerPatch := obj.HorizontalPodAutoscalerPatch()
	podSpecPatch := obj.PodSpecPatch()
	var behaviourStabilizationWindowSeconds int32 = 0
	if obj.Type() == pdoknlv3.ServiceTypeWFS {
		behaviourStabilizationWindowSeconds = 300
	}

	labels := obj.GetLabels()
	labels[AppLabelKey] = MapserverName
	if err := smoothoperatorutils.SetImmutableLabels(GetReconcilerClient(r), autoscaler, labels); err != nil {
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

	metrics := []autoscalingv2.MetricSpec{}
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

	if err := smoothoperatorutils.EnsureSetGVK(GetReconcilerClient(r), autoscaler, autoscaler); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, autoscaler, GetReconcilerScheme(r))
}

func GetBareConfigMapBlobDownload(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-init-scripts",
			Namespace: obj.GetNamespace(),
		},
	}
}

func MutateConfigMapBlobDownload[R Reconciler](r R, obj metav1.Object, configMap *corev1.ConfigMap) error {
	reconcilerClient := GetReconcilerClient(r)

	labels := smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels())
	labels[AppLabelKey] = MapserverName
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
	if err := ctrl.SetControllerReference(obj, configMap, GetReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func GetBareService(obj metav1.Object) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func MutateService[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, service *corev1.Service) error {
	reconcilerClient := GetReconcilerClient(r)

	labels := smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels())
	selector := smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels())
	selector[AppLabelKey] = MapserverName
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
		// options.disableWebserviceProxy not set or false
		if obj.Options() != nil && (obj.Options().DisableWebserviceProxy == nil || *obj.Options().DisableWebserviceProxy == false) {
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
	return ctrl.SetControllerReference(obj, service, GetReconcilerScheme(r))
}

func GetBareCorsHeadersMiddleware(obj metav1.Object) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + corsHeadersName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
			UID:       obj.GetUID(),
		},
	}
}

func MutateCorsHeadersMiddleware[R Reconciler](r R, obj metav1.Object, middleware *traefikiov1alpha1.Middleware) error {
	reconcilerClient := GetReconcilerClient(r)

	labels := smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels())
	labels[AppLabelKey] = MapserverName
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

	return ctrl.SetControllerReference(obj, middleware, GetReconcilerScheme(r))
}

func GetBarePodDisruptionBudget(obj metav1.Object) *v1.PodDisruptionBudget {
	return &v1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func MutatePodDisruptionBudget[R Reconciler](r R, obj metav1.Object, podDisruptionBudget *v1.PodDisruptionBudget) error {
	reconcilerClient := GetReconcilerClient(r)

	labels := smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels())
	labels[AppLabelKey] = MapserverName
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
	return ctrl.SetControllerReference(obj, podDisruptionBudget, GetReconcilerScheme(r))
}

func GetBareConfigMap(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-" + MapserverName,
			Namespace: obj.GetNamespace(),
		},
	}
}

func MutateConfigMap[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap) error {
	reconcilerClient := GetReconcilerClient(r)

	labels := smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels())
	labels[AppLabelKey] = MapserverName
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
	if err := ctrl.SetControllerReference(obj, configMap, GetReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func LogAndUpdateStatusError[R Reconciler](ctx context.Context, r R, obj client.Object, err error) {
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

func LogAndUpdateStatusFinished[R Reconciler](ctx context.Context, r R, obj client.Object, operationResults map[string]controllerutil.OperationResult) {
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
	if err := GetReconcilerClient(r).Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		log.FromContext(ctx).Error(err, "unable to update status")
		return
	}

	var status model.OperatorStatus
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		status = any(obj).(*pdoknlv3.WFS).Status
	case *pdoknlv3.WMS:
		status = any(obj).(*pdoknlv3.WMS).Status
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
	fmt.Println("Changed", status.OperationResults)
	if err := r.Status().Update(ctx, obj); err != nil {
		lgr.Error(err, "unable to update status")
	}
}
