package v3

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (wfs *WFS) ValidateCreate() ([]string, error) {
	warnings := []string{}
	allErrs := field.ErrorList{}

	err := sharedValidation.ValidateLabelsOnCreate(wfs.Labels)
	if err != nil {
		allErrs = append(allErrs, err)
	}

	err = sharedValidation.ValidateIngressRouteURLsContainsBaseURL(wfs.Spec.IngressRouteURLs, wfs.URL(), nil)
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

func (wfs *WFS) ValidateUpdate(wfsOld *WFS) ([]string, error) {
	return ValidateUpdate(wfs, wfsOld, ValidateWFS)
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

	if service.Mapfile == nil && service.DefaultCrs != "EPSG:28992" && service.Bbox == nil {
		*allErrs = append(*allErrs, field.Required(path.Child("bbox").Child("defaultCRS"), "when service.defaultCRS is not 'EPSG:28992'"))
	}

	if service.Mapfile != nil && service.Bbox != nil {
		sharedValidation.AddWarning(warnings, *path.Child("bbox"), "is not used when service.mapfile is configured", wfs.GroupVersionKind(), wfs.GetName())
	}

	podSpecPatch := wfs.Spec.PodSpecPatch
	ValidateEphemeralStorage(podSpecPatch, allErrs)
}

func ValidateEphemeralStorage(podSpecPatch corev1.PodSpec, allErrs *field.ErrorList) {
	path := field.NewPath("spec").
		Child("podSpecPatch").
		Child("containers").
		Key("mapserver").
		Child("resources").
		Child("limits").
		Child(corev1.ResourceEphemeralStorage.String())
	storageSet := false
	for _, container := range podSpecPatch.Containers {
		if container.Name == "mapserver" {
			_, storageSet = container.Resources.Limits[corev1.ResourceEphemeralStorage]
		}
	}
	if !storageSet {
		*allErrs = append(*allErrs, field.Required(path, ""))
	}
}
