package v3

import (
	"fmt"
	"strings"

	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"
)

func (wfs *WFS) ValidateCreate() ([]string, error) {
	warnings := []string{}
	reasons := []string{}

	err := sharedValidation.ValidateLabelsOnCreate(wfs.Labels)
	if err != nil {
		reasons = append(reasons, fmt.Sprintf("%v", err))
	}

	ValidateWFS(wfs, &warnings, &reasons)

	if len(reasons) > 0 {
		return warnings, fmt.Errorf("%s", strings.Join(reasons, ". "))
	}

	return warnings, nil
}

func (wfs *WFS) ValidateUpdate(wfsOld *WFS) ([]string, error) {
	warnings := []string{}
	reasons := []string{}

	// Check labels did not change
	err := sharedValidation.ValidateLabelsOnUpdate(wfsOld.Labels, wfs.Labels)
	if err != nil {
		reasons = append(reasons, fmt.Sprintf("%v", err))
	}

	sharedValidation.CheckBaseUrlImmutability(wfsOld, wfs, &reasons)

	if (wfs.Spec.Service.Inspire == nil && wfsOld.Spec.Service.Inspire != nil) || (wfs.Spec.Service.Inspire != nil && wfsOld.Spec.Service.Inspire == nil) {
		reasons = append(reasons, "services cannot change from inspire to not inspire or the other way around")
	}

	ValidateWFS(wfs, &warnings, &reasons)

	if len(reasons) > 0 {
		return warnings, fmt.Errorf("%s", strings.Join(reasons, ". "))
	}

	return warnings, nil
}

func ValidateWFS(wfs *WFS, warnings *[]string, reasons *[]string) {
	if strings.Contains(wfs.GetName(), "wfs") {
		*warnings = append(*warnings, sharedValidation.FormatValidationWarning("name should not contain wfs", wfs.GroupVersionKind(), wfs.GetName()))
	}

	service := wfs.Spec.Service

	err := sharedValidation.ValidateBaseURL(service.URL)
	if err != nil {
		*reasons = append(*reasons, fmt.Sprintf("%v", err))
	}

	if service.Mapfile == nil && service.DefaultCrs != "EPSG:28992" && service.Bbox == nil {
		*reasons = append(*reasons, "service.bbox.defaultCRS is required when service.defaultCRS is not 'EPSG:28992'")
	}

	if service.Mapfile != nil {
		if service.Bbox != nil {
			*warnings = append(*warnings, sharedValidation.FormatValidationWarning("service.bbox is not used when service.mapfile is configured", wfs.GroupVersionKind(), wfs.GetName()))
		}
	}
}
