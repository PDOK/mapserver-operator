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
	"github.com/pdok/mapserver-operator/internal/controller/blobdownload"
	"github.com/pdok/mapserver-operator/internal/controller/mapfilegenerator"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
)

const (
	appLabelKey           = "app"
	WFSName               = "WFS"
	downloadScriptName    = "gpkg_download.sh"
	mapfileGeneratorInput = "input.json"
	srvDir                = "/srv"
	blobsConfigName       = "blobsConfig"
	blobsSecretName       = "blobsSecret"
)

// WFSReconciler reconciles a WFS object
type WFSReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	MultitoolImage        string
	MapfileGeneratorImage string
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

	// Fetch the Atom instance
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

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

func getBareConfigMapBlobDownload(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getBareDeployment(obj).GetName() + "-init-scripts",
			Namespace: obj.GetNamespace(),
		},
	}
}

func getBareConfigMapMapfileGenerator(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getBareDeployment(obj).GetName() + "-mapfile-generator",
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *WFSReconciler) mutateConfigMapBlobDownload(WFS *pdoknlv3.WFS, configMap *corev1.ConfigMap) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(WFS.GetLabels())
	labels[appLabelKey] = WFSName
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		downloadScript, err := blobdownload.GetScript()
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{downloadScriptName: downloadScript}

	}
	configMap.Immutable = smoothoperatorutils.BoolPtr(true)

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(WFS, configMap, r.Scheme); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
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
	configMap.Immutable = smoothoperatorutils.BoolPtr(true)

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
func (r *WFSReconciler) mutateDeployment(wfs *pdoknlv3.WFS, deployment *appsv1.Deployment, blobDownloadConfigMapName string, mapfileGeneratorConfigMapName string) error {
	// Todo Mutate the other deployment parts, these are only the init-containers for blob-download and mapfile-generator
	podTemplateSpec := corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:            "blob-download",
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
					},
					Command: []string{"/bin/sh", "-c"},

					VolumeMounts: []corev1.VolumeMount{
						{Name: "base", MountPath: srvDir + "/data", ReadOnly: false},
						{Name: "data", MountPath: "/var/www", ReadOnly: false},
					},
				},
				{
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
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "mapfile-generator-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: mapfileGeneratorConfigMapName},
						},
					}},
			},
		},
	}

	if *wfs.Spec.Options.PrefetchData {
		mount := corev1.VolumeMount{
			Name:      "init-scripts",
			MountPath: "/src/scripts",
			ReadOnly:  true,
		}
		podTemplateSpec.Spec.Containers[0].VolumeMounts = append(podTemplateSpec.Spec.Containers[0].VolumeMounts, mount)
		volume := corev1.Volume{
			Name: "init-scripts",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: blobDownloadConfigMapName},
					DefaultMode:          smoothoperatorutils.Int32Ptr(0777),
				},
			},
		}
		podTemplateSpec.Spec.Volumes = append(podTemplateSpec.Spec.Volumes, volume)
	}

	args, err := blobdownload.GetArgs(*wfs)
	if err != nil {
		return err
	}
	podTemplateSpec.Spec.InitContainers[0].Args = []string{args}
	podTemplateSpec.Spec.InitContainers[0].Image = r.MultitoolImage

	resourceCPU := resource.MustParse("0.2")
	if useEphemeralVolume(wfs) {
		resourceCPU = resource.MustParse("1")
	}
	podTemplateSpec.Spec.InitContainers[0].Resources.Limits = corev1.ResourceList{
		corev1.ResourceCPU: resourceCPU,
	}

	deployment.Spec.Template = podTemplateSpec

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, deployment, deployment); err != nil {
		return err
	}
	return ctrl.SetControllerReference(wfs, deployment, r.Scheme)

}

// Use ephemeral volume when ephemeral storage is greater then 10Gi
func useEphemeralVolume(wfs *pdoknlv3.WFS) bool {
	threshold := resource.MustParse("10Gi")
	for _, container := range wfs.Spec.PodSpecPatch.Containers {
		if container.Name == "mapserver" {
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

// SetupWithManager sets up the controller with the Manager.
func (r *WFSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pdoknlv3.WFS{}).
		Named("wfs").
		Complete(r)
}
