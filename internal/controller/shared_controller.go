package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AppLabelKey               = "app"
	MapserverName             = "mapserver"
	LegendGeneratorName       = "legend-generator"
	FeatureInfoGeneratorName  = "featureinfo-generator"
	OgcWebserviceProxyName    = "ogc-webservice-proxy"
	MapfileGeneratorName      = "mapfile-generator"
	CapabilitiesGeneratorName = "capabilities-generator"
	InitScriptsName           = "init-scripts"
)

func ensureLabel[O pdoknlv3.WMSWFS](obj O, key, value string) {
	labels := obj.GetLabels()
	if _, ok := labels[key]; !ok {
		labels[key] = value
	}

	obj.SetLabels(labels)
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
