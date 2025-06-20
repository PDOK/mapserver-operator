package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	policyv1 "k8s.io/api/policy/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pdok/smooth-operator/model"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AppLabelKey     = "pdok.nl/app"
	InspireLabelKey = "pdok.nl/inspire"
)

func setWatches(mgr *builder.TypedBuilder[reconcile.Request]) *builder.TypedBuilder[reconcile.Request] {
	return mgr.Owns(&corev1.ConfigMap{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&traefikiov1alpha1.Middleware{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&traefikiov1alpha1.IngressRoute{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&policyv1.PodDisruptionBudget{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&smoothoperatorv1.OwnerInfo{}, builder.WithPredicates(predicate.GenerationChangedPredicate{}))
}

func ttlExpired[O pdoknlv3.WMSWFS](obj O) bool {
	var lifecycle *model.Lifecycle
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		wfs := any(obj).(*pdoknlv3.WFS)
		lifecycle = wfs.Spec.Lifecycle
	case *pdoknlv3.WMS:
		wms := any(obj).(*pdoknlv3.WMS)
		lifecycle = wms.Spec.Lifecycle
	}

	if lifecycle != nil && lifecycle.TTLInDays != nil {
		expiresAt := obj.GetCreationTimestamp().Add(time.Duration(*lifecycle.TTLInDays) * 24 * time.Hour)

		return expiresAt.Before(time.Now())
	}

	return false
}

func ensureLabel[O pdoknlv3.WMSWFS](obj O, key, value string) {
	labels := obj.GetLabels()
	if _, ok := labels[key]; !ok {
		labels[key] = value
	}

	obj.SetLabels(labels)
}

func getSuffixedName[O pdoknlv3.WMSWFS](obj O, suffix string) string {
	return obj.TypedName() + "-" + suffix
}

func addCommonLabels[O pdoknlv3.WMSWFS](obj O, labels map[string]string) map[string]string {
	labels[AppLabelKey] = constants.MapserverName

	inspire := false
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		inspire = any(obj).(*pdoknlv3.WFS).Spec.Service.Inspire != nil
	case *pdoknlv3.WMS:
		inspire = any(obj).(*pdoknlv3.WMS).Spec.Service.Inspire != nil
	}

	labels[InspireLabelKey] = strconv.FormatBool(inspire)

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
		if err != nil && !strings.Contains(err.Error(), "the object has been modified; please apply your changes to the latest version and try again") {
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
		err = createOrUpdateOrDeletePodDisruptionBudget(ctx, r, obj, operationResults)
		if err != nil {
			return operationResults, err
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
	configMaps[constants.MapserverName] = mutateConfigMap
	if obj.Mapfile() == nil {
		configMaps[constants.MapfileGeneratorName] = func(r R, o O, cm *corev1.ConfigMap) error {
			return mutateConfigMapMapfileGenerator(r, o, cm, ownerInfo)
		}
	}
	configMaps[constants.CapabilitiesGeneratorName] = func(r R, o O, cm *corev1.ConfigMap) error {
		return mutateConfigMapCapabilitiesGenerator(r, o, cm, ownerInfo)
	}
	if obj.Options().PrefetchData {
		configMaps[constants.InitScriptsName] = mutateConfigMapBlobDownload
	}
	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		wms, _ := any(obj).(*pdoknlv3.WMS)
		wmsReconciler := (*WMSReconciler)(r)

		configMaps[constants.LegendGeneratorName] = func(_ R, _ O, cm *corev1.ConfigMap) error {
			return mutateConfigMapLegendGenerator(wmsReconciler, wms, cm)
		}
		configMaps[constants.FeatureinfoGeneratorName] = func(_ R, _ O, cm *corev1.ConfigMap) error {
			return mutateConfigMapFeatureinfoGenerator(wmsReconciler, wms, cm)
		}
		configMaps[constants.OgcWebserviceProxyName] = func(_ R, _ O, cm *corev1.ConfigMap) error {
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
		case constants.MapserverName:
			hashedConfigMapNames.Mapserver = cm.Name
		case constants.MapfileGeneratorName:
			hashedConfigMapNames.MapfileGenerator = cm.Name
		case constants.CapabilitiesGeneratorName:
			hashedConfigMapNames.CapabilitiesGenerator = cm.Name
		case constants.InitScriptsName:
			hashedConfigMapNames.InitScripts = cm.Name
		case constants.LegendGeneratorName:
			hashedConfigMapNames.LegendGenerator = cm.Name
		case constants.FeatureinfoGeneratorName:
			hashedConfigMapNames.FeatureInfoGenerator = cm.Name
		case constants.OgcWebserviceProxyName:
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

func createOrUpdateOrDeletePodDisruptionBudget[O pdoknlv3.WMSWFS, R Reconciler](ctx context.Context, reconciler R, obj O, operationResults map[string]controllerutil.OperationResult) (err error) {
	reconcilerClient := getReconcilerClient(reconciler)
	podDisruptionBudget := getBarePodDisruptionBudget(obj)
	autoscalerPatch := obj.HorizontalPodAutoscalerPatch()
	if autoscalerPatch != nil && autoscalerPatch.MinReplicas != nil && autoscalerPatch.MaxReplicas != nil &&
		*autoscalerPatch.MinReplicas == 1 && *autoscalerPatch.MaxReplicas == 1 {
		err = reconcilerClient.Delete(ctx, podDisruptionBudget)
		if err == nil {
			operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, podDisruptionBudget)] = "deleted"
		}
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("unable to delete resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, podDisruptionBudget), err)
		}
	} else {
		operationResults[smoothoperatorutils.GetObjectFullName(reconcilerClient, podDisruptionBudget)], err = controllerutil.CreateOrUpdate(ctx, reconcilerClient, podDisruptionBudget, func() error {
			return mutatePodDisruptionBudget(reconciler, obj, podDisruptionBudget)
		})
		if err != nil {
			return fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(reconcilerClient, podDisruptionBudget), err)
		}
	}
	return nil
}

func recoveredPanicToError(rec any) (err error) {
	switch x := rec.(type) {
	case string:
		err = errors.New(x)
	case error:
		err = x
	default:
		err = errors.New("unknown panic")
	}

	// Add stack
	// TODO - this doesn't seem to work, see if there is a better method to add the stack
	err = errors.WithStack(err)

	return
}
