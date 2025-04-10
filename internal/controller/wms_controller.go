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
	"github.com/pdok/mapserver-operator/internal/controller/featureinfogenerator"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

const (
	featureinfoGeneratorInput = "input.json"
)

// WMSReconciler reconciles a WMS object
type WMSReconciler struct {
	client.Client
	Scheme                    *runtime.Scheme
	FeatureinfoGeneratorImage string
}

// +kubebuilder:rbac:groups=pdok.nl,resources=wms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pdok.nl,resources=wms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pdok.nl,resources=wms/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WMS object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *WMSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	// TODO use this featureinfoGeneratorInitContainer in the Reconcile process
	_, err := featureinfogenerator.GetFeatureinfoGeneratorInitContainer(r.FeatureinfoGeneratorImage, srvDir)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WMSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pdoknlv3.WMS{}).
		Named("wms").
		Complete(r)
}

func GetBareConfigMapFeatureinfoGenerator(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-featureinfo-generator",
			Namespace: obj.GetNamespace(),
		},
	}
}

func MutateConfigMapFeatureinfoGenerator(r WMSReconciler, wms *pdoknlv3.WMS, configMap *corev1.ConfigMap, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	labels := AddCommonLabels(wms, smoothoperatorutils.CloneOrEmptyMap(wms.GetLabels()))
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
