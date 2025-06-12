package v3

import (
	"slices"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (wfs *WFS) ValidateCreate(c client.Client) ([]string, error) {
	return ValidateCreate(c, wfs, ValidateWFS)
}

func (wfs *WFS) ValidateUpdate(c client.Client, wfsOld *WFS) ([]string, error) {
	return ValidateUpdate(c, wfs, wfsOld, ValidateWFS)
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
		*allErrs = append(*allErrs, field.Required(
			path.Child("bbox").Child("defaultCRS"),
			"when service.defaultCRS is not 'EPSG:28992'",
		))
	}

	if service.Mapfile != nil && service.Bbox != nil {
		sharedValidation.AddWarning(
			warnings,
			*path.Child("bbox"),
			"is not used when service.mapfile is configured",
			wfs.GroupVersionKind(),
			wfs.GetName(),
		)
	}

	crsses := []string{}
	for i, crs := range service.OtherCrs {
		if slices.Contains(crsses, crs) {
			*allErrs = append(*allErrs, field.Duplicate(
				path.Child("otherCrs").Index(i),
				crs,
			))
		} else {
			crsses = append(crsses, crs)
		}
	}

	ValidateInspire(wfs, allErrs)

	if wfs.Spec.HorizontalPodAutoscalerPatch != nil {
		ValidateHorizontalPodAutoscalerPatch(*wfs.Spec.HorizontalPodAutoscalerPatch, allErrs)
	}

	podSpecPatch := wfs.Spec.PodSpecPatch
	ValidateEphemeralStorage(podSpecPatch, allErrs)

	ValidateFeatureTypes(wfs, warnings, allErrs)
}

func ValidateFeatureTypes(wfs *WFS, warnings *[]string, allErrs *field.ErrorList) {
	names := []string{}
	path := field.NewPath("spec").Child("service").Child("featureTypes")
	for index, featureType := range wfs.Spec.Service.FeatureTypes {
		if slices.Contains(names, featureType.Name) {
			*allErrs = append(*allErrs, field.Duplicate(
				path.Index(index).Child("name"),
				featureType.Name,
			))
		} else {
			names = append(names, featureType.Name)
		}

		if wfs.Spec.Service.Mapfile != nil && featureType.Bbox != nil && featureType.Bbox.DefaultCRS != nil {
			sharedValidation.AddWarning(
				warnings,
				*path.Index(index).Child("bbox").Child("defaultCrs"),
				"is not used when service.mapfile is configured",
				wfs.GroupVersionKind(),
				wfs.GetName(),
			)
		}

	}
}
