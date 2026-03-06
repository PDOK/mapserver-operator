//nolint:dupl
package v3

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

// log is for logging in this package.
var wmsLog = logf.Log.WithName("wms-resource")

// SetupWMSWebhookWithManager registers the webhook for WMS in the manager.
func SetupWMSWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&pdoknlv3.WMS{}).
		WithValidator(&WMSCustomValidator{mgr.GetClient()}).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-pdok-nl-v3-wms,mutating=false,failurePolicy=fail,sideEffects=None,groups=pdok.nl,resources=wms,verbs=create;update,versions=v3,name=vwms-v3.kb.io,admissionReviewVersions=v1

// WMSCustomValidator struct is responsible for validating the WMS resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type WMSCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &WMSCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type WMS.
func (v *WMSCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	wms, ok := obj.(*pdoknlv3.WMS)
	if !ok {
		return nil, fmt.Errorf("expected a WMS object but got %T", obj)
	}
	wmsLog.Info("Validation for WMS upon creation", "name", wms.GetName())

	return wms.ValidateCreate(v.Client)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type WMS.
func (v *WMSCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	wms, ok := newObj.(*pdoknlv3.WMS)
	if !ok {
		return nil, fmt.Errorf("expected a WMS object for the newObj but got %T", newObj)
	}
	wmsOld, ok := oldObj.(*pdoknlv3.WMS)
	if !ok {
		return nil, fmt.Errorf("expected a WMS object for the oldObj but got %T", newObj)
	}
	wmsLog.Info("Validation for WMS upon update", "name", wms.GetName())

	return wms.ValidateUpdate(v.Client, wmsOld)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type WMS.
func (v *WMSCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	wms, ok := obj.(*pdoknlv3.WMS)
	if !ok {
		return nil, fmt.Errorf("expected a WMS object but got %T", obj)
	}
	wmsLog.Info("Validation for WMS upon deletion", "name", wms.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
