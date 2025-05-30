package v3

import (
	"slices"
	"strings"

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
		}

		if !slices.Contains(names, featureType.Name) {
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

		if tif := featureType.Data.TIF; tif != nil {
			if tif.Resample != "NEAREST" {
				sharedValidation.AddWarning(
					warnings,
					*path.Index(index).Child("data").Child("tif").Child("resample"),
					"is not used when service.mapfile is configured",
					wfs.GroupVersionKind(),
					wfs.GetName(),
				)
			}

			if tif.Offsite != nil {
				sharedValidation.AddWarning(
					warnings,
					*path.Index(index).Child("data").Child("tif").Child("offsite"),
					"is not used when service.mapfile is configured",
					wfs.GroupVersionKind(),
					wfs.GetName(),
				)
			}

			if tif.GetFeatureInfoIncludesClass {
				sharedValidation.AddWarning(
					warnings,
					*path.Index(index).Child("data").Child("tif").Child("getFeatureInfoIncludesClass"),
					"is not used when service.mapfile is configured",
					wfs.GroupVersionKind(),
					wfs.GetName(),
				)
			}
		}

	}
}
