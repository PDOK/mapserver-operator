//nolint:dupl
package v3

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

// log is for logging in this package.
var wfsLog = logf.Log.WithName("wfs-resource")

// SetupWFSWebhookWithManager registers the webhook for WFS in the manager.
func SetupWFSWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &pdoknlv3.WFS{}).
		WithValidator(&WFSCustomValidator{mgr.GetClient()}).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-pdok-nl-v3-wfs,mutating=false,failurePolicy=fail,sideEffects=None,groups=pdok.nl,resources=wfs,verbs=create;update,versions=v3,name=vwfs-v3.kb.io,admissionReviewVersions=v1

// WFSCustomValidator struct is responsible for validating the WFS resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type WFSCustomValidator struct {
	Client client.Client
}

var _ admission.Validator[*pdoknlv3.WFS] = &WFSCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type WFS.
func (v *WFSCustomValidator) ValidateCreate(_ context.Context, wfs *pdoknlv3.WFS) (admission.Warnings, error) {
	wfsLog.Info("Validation for WFS upon creation", "name", wfs.GetName())

	return wfs.ValidateCreate(v.Client)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type WFS.
func (v *WFSCustomValidator) ValidateUpdate(_ context.Context, wfsOld, wfs *pdoknlv3.WFS) (admission.Warnings, error) {
	wfsLog.Info("Validation for WFS upon update", "name", wfs.GetName())

	return wfs.ValidateUpdate(v.Client, wfsOld)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type WFS.
func (v *WFSCustomValidator) ValidateDelete(_ context.Context, wfs *pdoknlv3.WFS) (admission.Warnings, error) {
	wfsLog.Info("Validation for WFS upon deletion", "name", wfs.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
