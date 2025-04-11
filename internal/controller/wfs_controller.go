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
	"fmt"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/blobdownload"
	"github.com/pdok/mapserver-operator/internal/controller/capabilitiesgenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	"github.com/pdok/mapserver-operator/internal/controller/types"

	"github.com/pdok/mapserver-operator/internal/controller/mapfilegenerator"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	downloadScriptName    = "gpkg_download.sh"
	mapfileGeneratorInput = "input.json"
	srvDir                = "/srv"
	// TODO make dynamic?
	blobsConfigName = "blobs-9d7fcgcfcc"
	// TODO make dynamic?
	blobsSecretName            = "blobs-8ch6mbkg8t"
	capabilitiesGeneratorInput = "input.yaml"
	inputDir                   = "/input"
	postgisConfigName          = "postgisConfig"
	postgisSecretName          = "postgisSecret"
)

var (
	finalizerName = "wfs-controller" + "." + pdoknlv3.GroupVersion.Group + "/finalizer"
)

// WFSReconciler reconciles a WFS object
type WFSReconciler struct {
	client.Client
	Scheme                     *runtime.Scheme
	MapserverImage             string
	MultitoolImage             string
	MapfileGeneratorImage      string
	CapabilitiesGeneratorImage string
}

// +kubebuilder:rbac:groups=pdok.nl,resources=wfs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pdok.nl,resources=wfs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pdok.nl,resources=wfs/finalizers,verbs=update
// +kubebuilder:rbac:groups=pdok.nl,resources=ownerinfo,verbs=get;list;watch
// +kubebuilder:rbac:groups=pdok.nl,resources=ownerinfo/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=configmaps;services,verbs=watch;create;get;update;list;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=watch;create;get;update;list;delete
// +kubebuilder:rbac:groups=traefik.io,resources=ingressroutes;middlewares,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=create;update;delete;list;watch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets/status,verbs=get;update
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WFS object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *WFSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	lgr := log.FromContext(ctx)
	lgr.Info("Starting reconcile for WFS resource", "name", req.NamespacedName)

	// Fetch the WFS instance
	wfs := &pdoknlv3.WFS{}
	if err = r.Client.Get(ctx, req.NamespacedName, wfs); err != nil {
		if apierrors.IsNotFound(err) {
			lgr.Info("WFS resource not found", "name", req.NamespacedName)
		} else {
			lgr.Error(err, "unable to fetch WFS resource", "error", err)
		}
		return result, client.IgnoreNotFound(err)
	}

	lgr.Info("Fetching OwnerInfo", "name", req.NamespacedName)
	// Fetch the OwnerInfo instance
	ownerInfo := &smoothoperatorv1.OwnerInfo{}
	objectKey := client.ObjectKey{
		Namespace: "default", // wfs.Namespace,
		Name:      wfs.Spec.Service.OwnerInfoRef,
	}
	if err := r.Client.Get(ctx, objectKey, ownerInfo); err != nil {
		if apierrors.IsNotFound(err) {
			lgr.Info("OwnerInfo resource not found", "name", req.NamespacedName)
		} else {
			lgr.Error(err, "unable to fetch OwnerInfo resource", "error", err)
		}
		return result, client.IgnoreNotFound(err)
	}

	lgr.Info("Get object full name")
	fullName := smoothoperatorutils.GetObjectFullName(r.Client, wfs)
	lgr.Info("Finalize if necessary")
	shouldContinue, err := smoothoperatorutils.FinalizeIfNecessary(ctx, r.Client, wfs, finalizerName, func() error {
		lgr.Info("deleting resources", "name", fullName)
		return r.deleteAllForWFS(ctx, wfs, ownerInfo)
	})
	if !shouldContinue || err != nil {
		return result, err
	}

	lgr.Info("creating resources for wfs", "wfs", wfs)
	operationResults, err := r.createOrUpdateAllForWFS(ctx, wfs, ownerInfo)
	if err != nil {
		lgr.Info("failed creating resources for wfs", "wfs", wfs)
		LogAndUpdateStatusError(ctx, r, wfs, err)
		return result, err
	}
	lgr.Info("finished creating resources for wfs", "atom", wfs)
	LogAndUpdateStatusFinished(ctx, r, wfs, operationResults)

	return result, err
}

func (r *WFSReconciler) createOrUpdateAllForWFS(ctx context.Context, wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (operationResults map[string]controllerutil.OperationResult, err error) {
	operationResults = make(map[string]controllerutil.OperationResult)
	c := r.Client

	hashedConfigMapNames := types.HashedConfigMapNames{}

	// region ConfigMap
	{
		configMap := GetBareConfigMap(wfs)
		if err = MutateConfigMap(r, wfs, configMap); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, configMap)], err = controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
			return MutateConfigMap(r, wfs, configMap)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, configMap), err)
		}
		hashedConfigMapNames.ConfigMap = configMap.Name
	}
	// end region ConfigMap

	// region ConfigMap-MapfileGenerator
	{
		configMapMfg := GetBareConfigMapMapfileGenerator(wfs)
		if err = MutateConfigMapMapfileGenerator(r, wfs, configMapMfg, ownerInfo); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, configMapMfg)], err = controllerutil.CreateOrUpdate(ctx, r.Client, configMapMfg, func() error {
			return MutateConfigMapMapfileGenerator(r, wfs, configMapMfg, ownerInfo)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, configMapMfg), err)
		}
		hashedConfigMapNames.MapfileGenerator = configMapMfg.Name
	}
	// end region ConfigMap-MapfileGenerator

	// region ConfigMap-CapabilitieGenerator
	{
		configMapCg := GetBareConfigMapCapabilitiesGenerator(wfs)
		if err = MutateConfigMapCapabilitiesGenerator(r, wfs, configMapCg, ownerInfo); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, configMapCg)], err = controllerutil.CreateOrUpdate(ctx, r.Client, configMapCg, func() error {
			return MutateConfigMapCapabilitiesGenerator(r, wfs, configMapCg, ownerInfo)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, configMapCg), err)
		}
		hashedConfigMapNames.CapabilitiesGenerator = configMapCg.Name
	}
	// end region ConfigMap-CapabilitiesGenerator

	// region ConfigMap-BlobDownload
	{
		configMapBd := GetBareConfigMapBlobDownload(wfs)
		if err = MutateConfigMapBlobDownload(r, wfs, configMapBd); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, configMapBd)], err = controllerutil.CreateOrUpdate(ctx, r.Client, configMapBd, func() error {
			return MutateConfigMapBlobDownload(r, wfs, configMapBd)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, configMapBd), err)
		}
		hashedConfigMapNames.BlobDownload = configMapBd.Name
	}
	// end region ConfigMap-BlobDownload

	// region Deployment
	{
		deployment := mapserver.GetBareDeployment(wfs, MapserverName)
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, deployment)], err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
			return r.mutateDeployment(wfs, deployment, hashedConfigMapNames)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, deployment), err)
		}
	}
	// end region Deployment

	// region TraefikMiddleware
	{
		middleware := GetBareCorsHeadersMiddleware(wfs)
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, middleware)], err = controllerutil.CreateOrUpdate(ctx, r.Client, middleware, func() error {
			return MutateCorsHeadersMiddleware(r, wfs, middleware)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, middleware), err)
		}
	}
	// end region TraefikMiddleware

	// region PodDisruptionBudget
	{
		podDisruptionBudget := GetBarePodDisruptionBudget(wfs)
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, podDisruptionBudget)], err = controllerutil.CreateOrUpdate(ctx, r.Client, podDisruptionBudget, func() error {
			return MutatePodDisruptionBudget(r, wfs, podDisruptionBudget)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, podDisruptionBudget), err)
		}
	}
	// end region PodDisruptionBudget

	// region HorizontalAutoScaler
	{
		autoscaler := GetBareHorizontalPodAutoScaler(wfs)
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, autoscaler)], err = controllerutil.CreateOrUpdate(ctx, r.Client, autoscaler, func() error {
			return MutateHorizontalPodAutoscaler(r, wfs, autoscaler)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, autoscaler), err)
		}
	}
	// end region HorizontalAutoScaler

	// region IngressRoute
	{
		ingress := GetBareIngressRoute(wfs)
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, ingress)], err = controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
			return MutateIngressRoute(r, wfs, ingress)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, ingress), err)
		}
	}
	// end region IngressRoute

	// region Service
	{
		service := GetBareService(wfs)
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, service)], err = controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
			return MutateService(r, wfs, service)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, service), err)
		}
	}
	// end region Service

	return operationResults, nil
}

func (r *WFSReconciler) deleteAllForWFS(ctx context.Context, wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (err error) {
	bareObjects := GetSharedBareObjects(wfs)
	objects := []client.Object{}

	// Remove ConfigMaps as they have hashed names
	for _, object := range bareObjects {
		if _, ok := object.(*corev1.ConfigMap); !ok {
			objects = append(objects, object)
		}
	}

	// ConfigMap
	cm := GetBareConfigMap(wfs)
	err = MutateConfigMap(r, wfs, cm)
	if err != nil {
		return err
	}
	objects = append(objects, cm)

	// ConfigMap-MapfileGenerator
	cmMg := GetBareConfigMapMapfileGenerator(wfs)
	err = MutateConfigMapMapfileGenerator(r, wfs, cmMg, ownerInfo)
	if err != nil {
		return err
	}
	objects = append(objects, cmMg)

	// ConfigMap-CapabilitiesGenerator
	cmCg := GetBareConfigMapCapabilitiesGenerator(wfs)
	err = MutateConfigMapCapabilitiesGenerator(r, wfs, cmCg, ownerInfo)
	if err != nil {
		return err
	}
	objects = append(objects, cmCg)

	// ConfigMap-BlobDownload
	cmBd := GetBareConfigMapBlobDownload(wfs)
	err = MutateConfigMapBlobDownload(r, wfs, cmBd)
	if err != nil {
		return err
	}
	objects = append(objects, cmBd)

	return smoothoperatorutils.DeleteObjects(ctx, r.Client, objects)
}

// TODO Rename configMapnames
// TODO Make generic for WMS -> move to mapserver package
func (r *WFSReconciler) mutateDeployment(wfs *pdoknlv3.WFS, deployment *appsv1.Deployment, configMapNames types.HashedConfigMapNames) error {
	labels := AddCommonLabels(wfs, smoothoperatorutils.CloneOrEmptyMap(wfs.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, deployment, labels); err != nil {
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

	blobDownloadInitContainer, err := blobdownload.GetBlobDownloadInitContainer(wfs, r.MultitoolImage, blobsConfigName, blobsSecretName, srvDir)
	if err != nil {
		return err
	}
	mapfileGeneratorInitContainer, err := mapfilegenerator.GetMapfileGeneratorInitContainer(wfs, r.MapfileGeneratorImage, postgisConfigName, postgisSecretName, srvDir)
	if err != nil {
		return err
	}
	capabilitiesGeneratorInitContainer, err := capabilitiesgenerator.GetCapabilitiesGeneratorInitContainer(wfs, r.CapabilitiesGeneratorImage)
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
			InitContainers: []corev1.Container{
				*blobDownloadInitContainer,
				*mapfileGeneratorInitContainer,
				*capabilitiesGeneratorInitContainer,
			},
			Volumes: mapserver.GetVolumesForDeployment(wfs, configMapNames),
			Containers: []corev1.Container{
				{
					Name:            MapserverName,
					Image:           r.MapserverImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
					Env:          mapserver.GetEnvVarsForDeployment(wfs, blobsSecretName),
					VolumeMounts: mapserver.GetVolumeMountsForDeployment(wfs, srvDir),
					Resources:    mapserver.GetResourcesForDeployment(wfs),
					//LivenessProbe:  &corev1.Probe{}, // TODO
					//ReadinessProbe: &corev1.Probe{}, // TODO
					//StartupProbe:   &corev1.Probe{}, // TODO
					Lifecycle: &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{"sleep", "15"},
							},
							// Doesn't work
							//Sleep: &corev1.SleepAction{Seconds: 15},
						},
					},
				},
			},
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, deployment, deployment); err != nil {
		return err
	}
	return ctrl.SetControllerReference(wfs, deployment, r.Scheme)
}

// SetupWithManager sets up the controller with the Manager.
func (r *WFSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pdoknlv3.WFS{}).
		Named("wfs").
		Complete(r)
}
