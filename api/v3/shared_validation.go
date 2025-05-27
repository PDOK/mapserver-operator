package v3

import (
	"fmt"

	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"
	v1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateUpdate[W WMSWFS](newW, oldW W, validate func(W, *[]string, *field.ErrorList)) ([]string, error) {
	warnings := []string{}
	allErrs := field.ErrorList{}

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

func ValidateHorizontalPodAutoscalerPatch(patch HorizontalPodAutoscalerPatch, allErrs *field.ErrorList) {
	path := field.NewPath("spec").Child("horizontalPodAutoscaler")
	// TODO: replace hardcoded defaults with dynamic defaults from cli options or ownerInfo
	var minReplicas, maxReplicas int32 = 2, 32
	if patch.MinReplicas != nil {
		minReplicas = *patch.MinReplicas
	}
	if patch.MaxReplicas != nil {
		maxReplicas = *patch.MaxReplicas
	}

	if maxReplicas < minReplicas {
		replicas := fmt.Sprintf("minReplicas: %d, maxReplicas: %d", minReplicas, maxReplicas)

		*allErrs = append(*allErrs, field.Invalid(path, replicas, "maxReplicas cannot be less than minReplicas"))
	}

}

func ValidateEphemeralStorage(podSpecPatch v1.PodSpec, allErrs *field.ErrorList) {
	path := field.NewPath("spec").
		Child("podSpecPatch").
		Child("containers").
		Key("mapserver").
		Child("resources").
		Child("limits").
		Child(v1.ResourceEphemeralStorage.String())
	storageSet := false
	for _, container := range podSpecPatch.Containers {
		if container.Name == "mapserver" {
			_, storageSet = container.Resources.Limits[v1.ResourceEphemeralStorage]
		}
	}
	if !storageSet {
		*allErrs = append(*allErrs, field.Required(path, ""))
	}
}
