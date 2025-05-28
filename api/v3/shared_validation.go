package v3

import (
	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateUpdate[W WMSWFS](newW, oldW W, validate func(W, *[]string, *field.ErrorList)) ([]string, error) {
	warnings := []string{}
	allErrs := field.ErrorList{}

	// Check that the ingressRouteUrls contain the base url and no urls have been removed
	err := sharedValidation.ValidateIngressRouteURLsContainsBaseURL(newW.IngressRouteURLs(), newW.URL(), nil)
	if err != nil {
		allErrs = append(allErrs, err)
	}
	sharedValidation.ValidateIngressRouteURLsNotRemoved(oldW.IngressRouteURLs(), newW.IngressRouteURLs(), &allErrs, nil)

	sharedValidation.ValidateLabelsOnUpdate(oldW.GetLabels(), newW.GetLabels(), &allErrs)

	path := field.NewPath("spec").Child("service").Child("url")
	sharedValidation.CheckUrlImmutability(
		oldW.URL(),
		newW.URL(),
		&allErrs,
		path,
	)

	if (newW.Inspire() == nil && oldW.Inspire() != nil) || (newW.Inspire() != nil && oldW.Inspire() == nil) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("service").Child("inspire"), "cannot change from inspire to not inspire or the other way around"))
	}

	validate(newW, &warnings, &allErrs)

	if len(allErrs) == 0 {
		return warnings, nil
	}
	return warnings, apierrors.NewInvalid(
		newW.GroupKind(),
		newW.GetName(), allErrs)
}
