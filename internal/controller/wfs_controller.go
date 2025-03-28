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
	"github.com/pdok/mapserver-operator/internal/controller/mapfilegenerator"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	appLabelKey           = "app"
	WFSName               = "mapserver"
	downloadScriptName    = "gpkg_download.sh"
	mapfileGeneratorInput = "input.json"
	srvDir                = "/srv"
	blobsConfigName       = "blobsConfig"
	blobsSecretName       = "blobsSecret"
)

var (
	finalizerName = "wfs-controller" + "." + pdoknlv3.GroupVersion.Group + "/finalizer"
)

// WFSReconciler reconciles a WFS object
type WFSReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	MapserverImage		  string
	MultitoolImage        string
	MapfileGeneratorImage string
}

type HashedConfigMapNames struct {
	ConfigMap             string
	BlobDownload          string
	MapfileGenerator      string
	CapabilitiesGenerator string
}

// +kubebuilder:rbac:groups=pdok.nl,resources=wfs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pdok.nl,resources=wfs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pdok.nl,resources=wfs/finalizers,verbs=update
// +kubebuilder:rbac:groups=pdok.nl,resources=ownerinfo,verbs=get;list;watch
// +kubebuilder:rbac:groups=pdok.nl,resources=ownerinfo/status,verbs=get

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
		Namespace: wfs.Namespace,
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

	hashedConfigMapNames := HashedConfigMapNames{}

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
		configMapMfg := getBareConfigMapMapfileGenerator(wfs)
		if err = r.mutateConfigMapMapfileGenerator(wfs, configMapMfg, ownerInfo); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, configMapMfg)], err = controllerutil.CreateOrUpdate(ctx, r.Client, configMapMfg, func() error {
			return r.mutateConfigMapMapfileGenerator(wfs, configMapMfg, ownerInfo)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, configMapMfg), err)
		}
		hashedConfigMapNames.MapfileGenerator = configMapMfg.Name
	}
	// end region ConfigMap-MapfileGenerator

	// region ConfigMap-CapabilitieGenerator
	{
		configMapCg := getBareConfigMapCapabilitiesGenerator(wfs)
		if err = r.mutateConfigMapCapabilitiesGenerator(wfs, configMapCg, ownerInfo); err != nil {
			return operationResults, err
		}
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, configMapCg)], err = controllerutil.CreateOrUpdate(ctx, r.Client, configMapCg, func() error {
			return r.mutateConfigMapCapabilitiesGenerator(wfs, configMapCg, ownerInfo)
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
		deployment := getBareDeployment(wfs)
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
		ingress := getBareIngressRoute(wfs)
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, ingress)], err = controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
			return r.mutateIngressRoute(wfs, ingress)
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
	objects := append(GetSharedBareObjects(wfs), []client.Object{
		getBareDeployment(wfs),
		getBareIngressRoute(wfs),
		getBareConfigMapMapfileGenerator(wfs),
		getBareConfigMapCapabilitiesGenerator(wfs),
	}...)

	return smoothoperatorutils.DeleteObjects(ctx, r.Client, objects)
}

func getBareConfigMapMapfileGenerator(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-mapfile-generator",
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *WFSReconciler) mutateConfigMapMapfileGenerator(WFS *pdoknlv3.WFS, configMap *corev1.ConfigMap, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(WFS.GetLabels())
	labels[appLabelKey] = WFSName
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {

		mapfileGeneratorConfig, err := mapfilegenerator.GetConfig(WFS, ownerInfo)
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{mapfileGeneratorInput: mapfileGeneratorConfig}

	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(WFS, configMap, r.Scheme); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func getBareConfigMapCapabilitiesGenerator(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-capabilities-generator",
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *WFSReconciler) mutateConfigMapCapabilitiesGenerator(WFS *pdoknlv3.WFS, configMap *corev1.ConfigMap, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(WFS.GetLabels())
	labels[appLabelKey] = WFSName
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, configMap, labels); err != nil {
		return err
	}

	// TODO set data

	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(WFS, configMap, r.Scheme); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func getBareDeployment(obj metav1.Object) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + WFSName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

// TODO Rename configMapnames
func (r *WFSReconciler) mutateDeployment(wfs *pdoknlv3.WFS, deployment *appsv1.Deployment, configMapNames HashedConfigMapNames) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(wfs.GetLabels())
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, deployment, labels); err != nil {
		return err
	}

	matchLabels := smoothoperatorutils.CloneOrEmptyMap(labels)
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: matchLabels,
	}

	initContainers := []corev1.Container{
		r.getMapfileGeneratorInitContainer(),
	}
	volumes := []corev1.Volume{
		{
			Name: "mapfile-generator-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapNames.MapfileGenerator},
				},
			},
		},
	}

	blobDownloadInitContainer, err := r.getBlobDownloadInitContainer(wfs)
	if err != nil {
		return err
	}
	if *wfs.Spec.Options.PrefetchData {
		mount := corev1.VolumeMount{
			Name:      "init-scripts",
			MountPath: "/src/scripts",
			ReadOnly:  true,
		}
		blobDownloadInitContainer.VolumeMounts = append(blobDownloadInitContainer.VolumeMounts, mount)
		volume := corev1.Volume{
			Name: "init-scripts",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapNames.BlobDownload},
					DefaultMode:          smoothoperatorutils.Pointer(int32(0777)),
				},
			},
		}
		volumes = append(volumes, volume)
	}
	initContainers = append(initContainers, blobDownloadInitContainer)

	// Todo Mutate the other deployment parts, these are only the init-containers for blob-download and mapfile-generator
	deployment.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      labels,
		},
		Spec: corev1.PodSpec{
			InitContainers: initContainers,
			Volumes:        volumes,
			Containers: []corev1.Container{
				{
					Name: MapserverName,
					Image: r.MapserverImage,
					Env: []corev1.EnvVar{

					}
				},
			},
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, deployment, deployment); err != nil {
		return err
	}
	return ctrl.SetControllerReference(wfs, deployment, r.Scheme)
}

// Use ephemeral volume when ephemeral storage is greater then 10Gi
func useEphemeralVolume(wfs *pdoknlv3.WFS) bool {
	threshold := resource.MustParse("10Gi")
	for _, container := range wfs.Spec.PodSpecPatch.Containers {
		if container.Name == "configmap_files" {
			if container.Resources.Limits.StorageEphemeral() != nil {
				if container.Resources.Limits.StorageEphemeral().Value() > threshold.Value() {
					return true
				}
				return false
			}
		}
	}
	return false
}

func (r *WFSReconciler) getBlobDownloadInitContainer(wfs *pdoknlv3.WFS) (corev1.Container, error) {
	args, err := blobdownload.GetArgs(*wfs)
	if err != nil {
		return corev1.Container{}, err
	}

	resourceCPU := resource.MustParse("0.2")
	if useEphemeralVolume(wfs) {
		resourceCPU = resource.MustParse("1")
	}

	return corev1.Container{
		Name:            "blob-download",
		Image:           r.MultitoolImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		EnvFrom: []corev1.EnvFromSource{
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: blobsConfigName, // Todo add this ConfigMap
					},
				},
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: blobsSecretName, // Todo add this Secret
					},
				},
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("0.15"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resourceCPU,
			},
		},
		Command: []string{"/bin/sh", "-c"},
		Args:    []string{args},

		VolumeMounts: []corev1.VolumeMount{
			{Name: "base", MountPath: srvDir + "/data", ReadOnly: false},
			{Name: "data", MountPath: "/var/www", ReadOnly: false},
		},
	}, nil
}

func (r *WFSReconciler) getMapfileGeneratorInitContainer() corev1.Container {
	return corev1.Container{
		Name:            "mapfile-generator",
		Image:           r.MapfileGeneratorImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"generate-mapfile"},
		Args: []string{ // todo
			"",
			"",
			"",
			"",
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "base", MountPath: srvDir + "/data", ReadOnly: false},
			{Name: "mapfile-generator-config", MountPath: "/input", ReadOnly: true},
		},
	}
}

func getBareIngressRoute(obj metav1.Object) *traefikiov1alpha1.IngressRoute {
	return &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *WFSReconciler) mutateIngressRoute(wfs *pdoknlv3.WFS, ingressRoute *traefikiov1alpha1.IngressRoute) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(wfs.GetLabels())
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, ingressRoute, labels); err != nil {
		return err
	}

	ingressRoute.Spec = traefikiov1alpha1.IngressRouteSpec{
		Routes: []traefikiov1alpha1.Route{
			{
				Kind:  "Rule",
				Match: getMatchRule(wfs),
				Services: []traefikiov1alpha1.Service{
					{
						LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
							Name: GetBareService(wfs).GetName(),
							Kind: "Service",
							Port: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: int32(MapserverPortNr),
							},
						},
					},
				},
				Middlewares: []traefikiov1alpha1.MiddlewareRef{
					{
						Name:      wfs.Name + "-" + corsHeadersName,
						Namespace: wfs.GetNamespace(),
					},
				},
			},
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, ingressRoute, ingressRoute); err != nil {
		return err
	}
	return ctrl.SetControllerReference(wfs, ingressRoute, r.Scheme)
}

func getMatchRule(wfs *pdoknlv3.WFS) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && Path(`/" + pdoknlv3.GetBaseURLPath(wfs) + "`)"
}

// SetupWithManager sets up the controller with the Manager.
func (r *WFSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pdoknlv3.WFS{}).
		Named("wfs").
		Complete(r)
}
