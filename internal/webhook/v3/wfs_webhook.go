/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v3

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

// nolint:unused
// log is for logging in this package.
var wfslog = logf.Log.WithName("wfs-resource")

// SetupWFSWebhookWithManager registers the webhook for WFS in the manager.
func SetupWFSWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&pdoknlv3.WFS{}).
		WithValidator(&WFSCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-pdok-nl-v3-wfs,mutating=false,failurePolicy=fail,sideEffects=None,groups=pdok.nl,resources=wfs,verbs=create;update,versions=v3,name=vwfs-v3.kb.io,admissionReviewVersions=v1

// WFSCustomValidator struct is responsible for validating the WFS resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type WFSCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &WFSCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type WFS.
func (v *WFSCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	wfs, ok := obj.(*pdoknlv3.WFS)
	if !ok {
		return nil, fmt.Errorf("expected a WFS object but got %T", obj)
	}
	wfslog.Info("Validation for WFS upon creation", "name", wfs.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type WFS.
func (v *WFSCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	wfs, ok := newObj.(*pdoknlv3.WFS)
	if !ok {
		return nil, fmt.Errorf("expected a WFS object for the newObj but got %T", newObj)
	}
	wfslog.Info("Validation for WFS upon update", "name", wfs.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type WFS.
func (v *WFSCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	wfs, ok := obj.(*pdoknlv3.WFS)
	if !ok {
		return nil, fmt.Errorf("expected a WFS object but got %T", obj)
	}
	wfslog.Info("Validation for WFS upon deletion", "name", wfs.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
