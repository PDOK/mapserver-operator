package v3

import (
	"github.com/pdok/smooth-operator/model"
	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateUpdate[W WMSWFS](newW, oldW W, validate func(W, *[]string, *field.ErrorList)) ([]string, error) {
	warnings := []string{}
	allErrs := field.ErrorList{}

	sharedValidation.ValidateLabelsOnUpdate(oldW.GetLabels(), newW.GetLabels(), &allErrs)

	path := field.NewPath("spec").Child("service").Child("url")
	oldURL, err := model.ParseURL(oldW.URLPath())
	if err != nil {
		allErrs = append(allErrs, field.InternalError(path, err))
	}
	newURL, err := model.ParseURL(oldW.URLPath())
	if err != nil {
		allErrs = append(allErrs, field.InternalError(path, err))
	}
	sharedValidation.CheckUrlImmutability(
		model.URL{URL: oldURL},
		model.URL{URL: newURL},
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
