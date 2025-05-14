package v3

import (
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"
)

func (wfs *WFS) ValidateCreate() ([]string, error) {
	warnings := []string{}
	allErrs := field.ErrorList{}

	err := sharedValidation.ValidateLabelsOnCreate(wfs.Labels)
	if err != nil {
		allErrs = append(allErrs, err)
	}

	ValidateWFS(wfs, &warnings, &allErrs)

	if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, apierrors.NewInvalid(
		schema.GroupKind{Group: "pdok.nl", Kind: "WFS"},
		wfs.Name, allErrs)
}

// TODO fix linting (dupl)
func (wfs *WFS) ValidateUpdate(wfsOld *WFS) ([]string, error) {
	warnings := []string{}
	allErrs := field.ErrorList{}

	sharedValidation.ValidateLabelsOnUpdate(wfsOld.Labels, wfs.Labels, &allErrs)

	sharedValidation.CheckBaseUrlImmutability(wfsOld, wfs, &allErrs)

	if (wfs.Spec.Service.Inspire == nil && wfsOld.Spec.Service.Inspire != nil) || (wfs.Spec.Service.Inspire != nil && wfsOld.Spec.Service.Inspire == nil) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("service").Child("inspire"), "cannot change from inspire to not inspire or the other way around"))
	}

	ValidateWFS(wfs, &warnings, &allErrs)

	if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, apierrors.NewInvalid(
		schema.GroupKind{Group: "pdok.nl", Kind: "WFS"},
		wfs.Name, allErrs)
}

func ValidateWFS(wfs *WFS, warnings *[]string, allErrs *field.ErrorList) {
	if strings.Contains(wfs.GetName(), "wfs") {
		sharedValidation.AddWarning(
			warnings,
			*field.NewPath("metadata").Child("name"),
			"name should not contain wfs",
			wfs.GroupVersionKind(),
			wfs.GetName(),
		)
	}

	service := wfs.Spec.Service
	path := field.NewPath("spec").Child("service")

	err := sharedValidation.ValidateBaseURL(service.URL)
	if err != nil {
		*allErrs = append(*allErrs, field.Invalid(path.Child("url"), service.URL, err.Error()))
	}

	if service.Mapfile == nil && service.DefaultCrs != "EPSG:28992" && service.Bbox == nil {
		*allErrs = append(*allErrs, field.Required(path.Child("bbox").Child("defaultCRS"), "when service.defaultCRS is not 'EPSG:28992'"))
	}

	if service.Mapfile != nil && service.Bbox != nil {
		sharedValidation.AddWarning(warnings, *path.Child("bbox"), "is not used when service.mapfile is configured", wfs.GroupVersionKind(), wfs.GetName())
	}
}
