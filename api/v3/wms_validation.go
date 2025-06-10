package v3

import (
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/strings/slices"
)

func (wms *WMS) ValidateCreate(c client.Client) ([]string, error) {
	return ValidateCreate(c, wms, ValidateWMS)
}

func (wms *WMS) ValidateUpdate(c client.Client, wmsOld *WMS) ([]string, error) {
	return ValidateUpdate(c, wms, wmsOld, ValidateWMS)
}

func ValidateWMS(wms *WMS, warnings *[]string, allErrs *field.ErrorList) {
	if strings.Contains(wms.GetName(), "wms") {
		sharedValidation.AddWarning(
			warnings,
			*field.NewPath("metadata").Child("name"),
			"name should not contain wms",
			wms.GroupVersionKind(),
			wms.GetName(),
		)
	}

	if wms.Mapfile() != nil {
		service := wms.Spec.Service
		path := field.NewPath("spec").Child("service")
		if service.Resolution != nil {
			sharedValidation.AddWarning(
				warnings,
				*path.Child("resolution"),
				"not used when service.mapfile is configured",
				wms.GroupVersionKind(),
				wms.GetName(),
			)
		}
		if service.DefResolution != nil {
			sharedValidation.AddWarning(
				warnings,
				*path.Child("defResolution"),
				"not used when service.mapfile is configured",
				wms.GroupVersionKind(),
				wms.GetName(),
			)
		}
	}

	ValidateInspire(wms, allErrs)
	if wms.HorizontalPodAutoscalerPatch() != nil {
		ValidateHorizontalPodAutoscalerPatch(*wms.HorizontalPodAutoscalerPatch(), allErrs)
	}
	ValidateEphemeralStorage(wms.PodSpecPatch(), allErrs)

	validateLayers(wms, warnings, allErrs)
}

func validateLayers(wms *WMS, warnings *[]string, allErrs *field.ErrorList) {

	layerNames := []string{}
	hasVisibleLayer := false

	topLayer := AnnotatedLayer{
		GroupName:    nil,
		IsTopLayer:   true,
		IsGroupLayer: true,
		IsDataLayer:  false,
		Layer:        wms.Spec.Service.Layer,
	}

	validateLayer(topLayer, field.NewPath("spec").Child("service").Child("layer"), []string{}, &layerNames, &hasVisibleLayer, wms, warnings, allErrs)

	if !hasVisibleLayer {
		*allErrs = append(*allErrs, field.Required(
			field.NewPath("spec").Child("service").Child("layer").Child("layers[*]").Child("visible"),
			"at least one layer must be visible",
		))
	}
}

func validateLayer(layer AnnotatedLayer, path *field.Path, groupStyles []string, layerNames *[]string, hasVisibleLayer *bool, wms *WMS, warnings *[]string, allErrs *field.ErrorList) {
	service := wms.Spec.Service

	var layerName string
	if layer.IsTopLayer && layer.Name == nil {
		layerName = "unnamed: " + TopLayer
	} else {
		layerName = *layer.Name
	}

	if slices.Contains(*layerNames, layerName) {
		*allErrs = append(*allErrs, field.Duplicate(
			path.Child("name"),
			layerName,
		))
	} else {
		*layerNames = append(*layerNames, layerName)
	}

	if layer.IsGroupLayer && layer.Data != nil {
		*allErrs = append(*allErrs, field.Invalid(
			path.Child("data"),
			layer.Data,
			"must not be set on a GroupLayer",
		))
	}

	validateLayerWithMapfile(layer, path, wms, warnings, allErrs)

	if layer.Visible {
		if !layer.IsTopLayer {
			*hasVisibleLayer = true
		}
	} else {
		validateNotVisibleLayer(layer, path, wms, warnings, allErrs)
	}

	styleNames := []string{}
	for i, style := range layer.Styles {
		stylePath := path.Child("styles").Index(i)
		validateStyle(style, stylePath, &styleNames, &groupStyles, service.StylingAssets.GetAllConfigMapRefKeys(), layer, service.Mapfile != nil, allErrs)
	}

	if layer.IsDataLayer {
		for _, groupStyle := range groupStyles {
			if !slices.Contains(styleNames, groupStyle) {
				*allErrs = append(*allErrs, field.Invalid(
					path.Child("styles"),
					nil,
					fmt.Sprintf("dataLayer must implement style: %s, defined by a parent layer", groupStyle),
				))
			}
		}
	}

	for i, subLayer := range layer.Layers {
		annotatedSubLayer := AnnotatedLayer{
			GroupName:    layer.Name,
			IsTopLayer:   false,
			IsGroupLayer: subLayer.IsGroupLayer(),
			IsDataLayer:  subLayer.IsDataLayer(),
			Layer:        subLayer,
		}
		validateLayer(annotatedSubLayer, path.Child("layers").Index(i), groupStyles, layerNames, hasVisibleLayer, wms, warnings, allErrs)
	}

}

func validateLayerWithMapfile(layer AnnotatedLayer, path *field.Path, wms *WMS, warnings *[]string, allErrs *field.ErrorList) {
	service := wms.Spec.Service
	hasCustomMapfile := service.Mapfile != nil
	if hasCustomMapfile && layer.BoundingBoxes != nil {
		sharedValidation.AddWarning(
			warnings,
			*path.Child("boundingBoxes"),
			"is not used when service.mapfile is configured",
			wms.GroupVersionKind(),
			wms.GetName(),
		)
	}
	if !hasCustomMapfile && service.DataEPSG != "EPSG:28992" && !layer.hasBoundingBoxForCRS(service.DataEPSG) && layer.Name != nil {
		*allErrs = append(*allErrs, field.Required(
			path.Child("boundingBoxes").Child("crs"),
			fmt.Sprintf("must contain a boundingBox for CRS %s when service.dataEPSG is not 'EPSG:28992'", service.DataEPSG),
		))
	}

	if layer.IsDataLayer && hasCustomMapfile {
		if tif := layer.Data.TIF; tif != nil {
			tifWarnings(tif, path, wms, warnings)
		}
	}

}

func tifWarnings(tif *TIF, path *field.Path, wms *WMS, warnings *[]string) {
	if tif.Resample != "NEAREST" {
		sharedValidation.AddWarning(
			warnings,
			*path.Child("data").Child("tif").Child("resample"),
			"is not used when service.mapfile is configured",
			wms.GroupVersionKind(),
			wms.GetName(),
		)
	}

	if tif.Offsite != nil {
		sharedValidation.AddWarning(
			warnings,
			*path.Child("data").Child("tif").Child("offsite"),
			"is not used when service.mapfile is configured",
			wms.GroupVersionKind(),
			wms.GetName(),
		)
	}

	if tif.GetFeatureInfoIncludesClass {
		sharedValidation.AddWarning(
			warnings,
			*path.Child("data").Child("tif").Child("getFeatureInfoIncludesClass"),
			"is not used when service.mapfile is configured",
			wms.GroupVersionKind(),
			wms.GetName(),
		)
	}
}

func validateStyle(style Style, path *field.Path, styleNames *[]string, groupStyles *[]string, stylingFiles []string, layer AnnotatedLayer, usesCustomMapfile bool, allErrs *field.ErrorList) {
	if slices.Contains(*styleNames, style.Name) {
		*allErrs = append(*allErrs, field.Invalid(
			path.Child("name"),
			style.Name,
			"A Layer can't use the same style name multiple times",
		))
	} else {
		*styleNames = append(*styleNames, style.Name)
	}

	if layer.Visible && !slices.Contains(*groupStyles, style.Name) && style.Title == nil {
		*allErrs = append(*allErrs, field.Required(
			path.Child("title"),
			"A Style must have a title on the highest visible Layer",
		))
	}

	if layer.IsGroupLayer {
		if slices.Contains(*groupStyles, style.Name) {
			*allErrs = append(*allErrs, field.Invalid(
				path.Child("name"),
				style.Name,
				"A GroupLayer can't redefine the same style as a parent layer",
			))
		} else {
			*groupStyles = append(*groupStyles, style.Name)
		}

		if style.Visualization != nil {
			*allErrs = append(*allErrs, field.Invalid(
				path.Child("visualization"),
				style.Visualization,
				"GroupLayers must not have a visualization",
			))
		}
	}

	if layer.IsDataLayer {
		switch {
		case usesCustomMapfile && style.Visualization != nil:
			*allErrs = append(*allErrs, field.Invalid(
				path.Child("visualization"),
				style.Visualization,
				"is not used when spec.service.mapfile is used",
			))
		case !usesCustomMapfile && style.Visualization == nil:
			*allErrs = append(*allErrs, field.Required(
				path.Child("visualization"),
				"on DataLayers when spec.service.mapfile is not used",
			))
		case !usesCustomMapfile && !slices.Contains(stylingFiles, *style.Visualization):
			*allErrs = append(*allErrs, field.Invalid(
				path.Child("visualization"),
				style.Visualization,
				"must be defined be in spec.service.stylingAssets.configMapKeyRefs.Keys",
			))
		}

	}
}

func validateNotVisibleLayer(layer AnnotatedLayer, path *field.Path, wms *WMS, warnings *[]string, allErrs *field.ErrorList) {
	if layer.IsGroupLayer {
		*allErrs = append(*allErrs, field.Invalid(
			path.Child("visible"),
			layer.Visible,
			"must be true for a "+GroupLayer,
		))
	}
	paths := []field.Path{}

	if layer.Title != nil {
		paths = append(paths, *path.Child("title"))
	}
	if layer.Abstract != nil {
		paths = append(paths, *path.Child("abstract"))
	}
	if layer.Keywords != nil {
		paths = append(paths, *path.Child("keywords"))
	}
	if layer.DatasetMetadataURL != nil {
		paths = append(paths, *path.Child("datasetMetadataURL"))
	}
	if layer.Authority != nil {
		paths = append(paths, *path.Child("authority"))
	}

	for i, style := range layer.Styles {
		if style.Title != nil {
			paths = append(paths, *path.Child("styles").Index(i).Child("title"))
		}
		if style.Abstract != nil {
			paths = append(paths, *path.Child("styles").Index(i).Child("abstract"))
		}
	}

	for _, path := range paths {
		sharedValidation.AddWarning(
			warnings,
			path,
			"is not used when layer.visible=false",
			wms.GroupVersionKind(),
			wms.GetName(),
		)
	}

}
