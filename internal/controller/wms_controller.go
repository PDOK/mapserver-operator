/*
MIT License

Copyright (c) 2024 Publieke Dienstverlening op de Kaart

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"context"

	"github.com/pdok/mapserver-operator/internal/controller/types"

	"github.com/pdok/mapserver-operator/internal/controller/featureinfogenerator"
	"github.com/pdok/mapserver-operator/internal/controller/legendgenerator"
	"github.com/pdok/mapserver-operator/internal/controller/ogcwebserviceproxy"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

const (
	ogcWebserviceProxyInput   = "service-config.yaml"
	featureinfoGeneratorInput = "input.json"
)

// WMSReconciler reconciles a WMS object
type WMSReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Images types.Images
}

// +kubebuilder:rbac:groups=pdok.nl,resources=wms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pdok.nl,resources=wms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pdok.nl,resources=wms/finalizers,verbs=update
// +kubebuilder:rbac:groups=pdok.nl,resources=ownerinfo,verbs=get;list;watch
// +kubebuilder:rbac:groups=pdok.nl,resources=ownerinfo/status,verbs=
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=configmaps;services,verbs=watch;create;get;update;list;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=watch;list;get
// +kubebuilder:rbac:groups=traefik.io,resources=ingressroutes;middlewares,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=watch;create;get;update;list;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=create;update;delete;list;watch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets/status,verbs=get;update
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// The Reconcile function compares the state specified by
// the WMS object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *WMSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	lgr := log.FromContext(ctx)
	lgr.Info("Starting reconcile for WMS resource", "name", req.NamespacedName)

	// Fetch the WMS instance
	wms := &pdoknlv3.WMS{}
	if err = r.Client.Get(ctx, req.NamespacedName, wms); err != nil {
		if apierrors.IsNotFound(err) {
			lgr.Info("WMS resource not found", "name", req.NamespacedName)
		} else {
			lgr.Error(err, "unable to fetch WMS resource", "error", err)
		}
		return result, client.IgnoreNotFound(err)
	}

	lgr.Info("Fetching OwnerInfo", "name", req.NamespacedName)
	// Fetch the OwnerInfo instance
	ownerInfo := &smoothoperatorv1.OwnerInfo{}
	objectKey := client.ObjectKey{
		Namespace: wms.Namespace,
		Name:      wms.Spec.Service.OwnerInfoRef,
	}
	if err := r.Client.Get(ctx, objectKey, ownerInfo); err != nil {
		if apierrors.IsNotFound(err) {
			lgr.Info("OwnerInfo resource not found", "name", req.NamespacedName)
		} else {
			lgr.Error(err, "unable to fetch OwnerInfo resource", "error", err)
		}
		return result, client.IgnoreNotFound(err)
	}

	ensureLabel(wms, "pdok.nl/service-type", "wms")

	lgr.Info("creating resources for wms", "wms", wms)
	operationResults, err := createOrUpdateAllForWMSWFS(ctx, r, wms, ownerInfo)
	if err != nil {
		lgr.Info("failed creating resources for wms", "wms", wms)
		logAndUpdateStatusError(ctx, r, wms, err)
		return result, err
	}
	lgr.Info("finished creating resources for wms", "wms", wms)
	logAndUpdateStatusFinished(ctx, r, wms, operationResults)

	return result, err
}

func mutateConfigMapLegendGenerator(r *WMSReconciler, wms *pdoknlv3.WMS, configMap *corev1.ConfigMap) error {
	labels := addCommonLabels(wms, smoothoperatorutils.CloneOrEmptyMap(wms.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		configMap.Data = legendgenerator.GetConfigMapData(wms)
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(wms, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)

}

func mutateConfigMapFeatureinfoGenerator(r *WMSReconciler, wms *pdoknlv3.WMS, configMap *corev1.ConfigMap) error {
	labels := addCommonLabels(wms, smoothoperatorutils.CloneOrEmptyMap(wms.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		input, err := featureinfogenerator.GetInput(wms)
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{featureinfoGeneratorInput: input}
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(wms, configMap, r.Scheme); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func mutateConfigMapOgcWebserviceProxy(r *WMSReconciler, wms *pdoknlv3.WMS, configMap *corev1.ConfigMap) error {

	labels := addCommonLabels(wms, smoothoperatorutils.CloneOrEmptyMap(wms.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		input, err := ogcwebserviceproxy.GetConfig(wms)
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{ogcWebserviceProxyInput: input}
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(wms, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

// SetupWithManager sets up the controller with the Manager.
func (r *WMSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pdoknlv3.WMS{}).
		Named("wms").
		Complete(r)
}
