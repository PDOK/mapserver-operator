package v3

import (
	"fmt"
	"maps"
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

// TODO fix linting (cyclop,funlen)
//
//nolint:cyclop,funlen
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

	service := wms.Spec.Service
	path := field.NewPath("spec").Child("service")

	if service.Mapfile != nil {
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

	var rewriteGroupToDataLayers = wms.Options().RewriteGroupToDataLayers
	var validateChildStyleNameEqual = wms.Options().ValidateChildStyleNameEqual

	equalChildStyleNames := map[string][]string{}
	if rewriteGroupToDataLayers && validateChildStyleNameEqual {
		findEqualChildStyleNames(&wms.Spec.Service.Layer, &equalChildStyleNames)
	}

	var names []string
	hasVisibleLayer := false
	wms.Spec.Service.Layer.setInheritedBoundingBoxes()
	for _, layer := range wms.Spec.Service.GetAllLayers() {
		path = path.Child("layers")
		var layerErrs field.ErrorList

		layerType := layer.GetLayerType(&service)
		var layerName string
		if layer.Name == nil {
			if layerType != TopLayer {
				layerErrs = append(layerErrs, field.Required(
					path.Child("[*]").Child("name"),
					"(except for the topLayer)",
				))
			}
			layerName = "unnamed:" + layerType
		} else {
			layerName = *layer.Name
		}

		path = path.Child(fmt.Sprintf("[%s]", layerName))

		if slices.Contains(names, layerName) {
			layerErrs = append(layerErrs, field.Duplicate(
				path,
				layerName,
			))
		}
		names = append(names, layerName)

		if service.Mapfile != nil && layer.BoundingBoxes != nil {
			sharedValidation.AddWarning(
				warnings,
				*path.Child("boundingBoxes"),
				"is not used when service.mapfile is configured",
				wms.GroupVersionKind(),
				wms.GetName(),
			)
		}
		if service.Mapfile == nil && service.DataEPSG != "EPSG:28992" && !layer.hasBoundingBoxForCRS(service.DataEPSG) {
			layerErrs = append(layerErrs, field.Required(
				path.Child("boundingBoxes").Child("crs"),
				fmt.Sprintf("must contain a boundingBox for CRS %s when service.dataEPSG is not 'EPSG:28992'", service.DataEPSG),
			))
		}

		//nolint:nestif
		if !layer.Visible {
			if layer.Title != nil {
				sharedValidation.AddWarning(
					warnings,
					*path.Child("title"),
					"is not used when layer.visible=false",
					wms.GroupVersionKind(),
					wms.GetName(),
				)
			}
			if layer.Abstract != nil {
				sharedValidation.AddWarning(
					warnings,
					*path.Child("abstrct"),
					"is not used when layer.visible=false",
					wms.GroupVersionKind(),
					wms.GetName(),
				)
			}
			if layer.Keywords != nil {
				sharedValidation.AddWarning(
					warnings,
					*path.Child("keywords"),
					"is not used when layer.visible=false",
					wms.GroupVersionKind(),
					wms.GetName(),
				)
			}
			if layer.DatasetMetadataURL != nil {
				sharedValidation.AddWarning(
					warnings,
					*path.Child("datasetMetadataURL"),
					"is not used when layer.visible=false",
					wms.GroupVersionKind(),
					wms.GetName(),
				)
			}
			if layer.Authority != nil && layer.Authority.SpatialDatasetIdentifier != "" {
				sharedValidation.AddWarning(
					warnings,
					*path.Child("authority").Child("spatialDatasetIdentifier"),
					"is not used when layer.visible=false",
					wms.GroupVersionKind(),
					wms.GetName(),
				)
			}

			for i, style := range layer.Styles {
				if style.Title != nil {
					sharedValidation.AddWarning(
						warnings,
						*path.Child("styles").Index(i).Child("title"),
						"is not used when layer.visible=false",
						wms.GroupVersionKind(),
						wms.GetName(),
					)
				}
				if style.Abstract != nil {
					sharedValidation.AddWarning(
						warnings,
						*path.Child("styles").Index(i).Child("abstract"),
						"is not used when layer.visible=false",
						wms.GroupVersionKind(),
						wms.GetName(),
					)
				}
			}
		}

		// TODO fix linting (nestif)
		//nolint:nestif
		if layer.Visible {
			hasVisibleLayer = true

			if layer.Title == nil {
				layerErrs = append(layerErrs, field.Required(path.Child("title"), "required if layer.visible=true"))
			}
			if layer.Abstract == nil {
				layerErrs = append(layerErrs, field.Required(path.Child("abstract"), "required if layer.visible=true"))
			}
			if layer.Keywords == nil {
				layerErrs = append(layerErrs, field.Required(path.Child("keywords"), "required if layer.visible=true"))
			}
			for i, style := range layer.Styles {
				if style.Title == nil {
					layerErrs = append(layerErrs, field.Required(
						path.Child("styles").Index(i),
						"required if layer.visible=true",
					))
				}
			}
			if !rewriteGroupToDataLayers && validateChildStyleNameEqual {
				equalStylesNames, ok := equalChildStyleNames[layerName]
				if ok {
					for _, styleName := range equalStylesNames {
						layerErrs = append(layerErrs, field.Invalid(
							path.Child("styles").Child(styleName),
							styleName,
							"style.name from parent layer must not be set on a a child layer style",
						))
					}
				}
			}
		}

		if layer.IsDataLayer() {
			for i, style := range layer.Styles {
				if wms.Spec.Service.Mapfile == nil && style.Visualization == nil {
					layerErrs = append(layerErrs, field.Required(
						path.Child("styles").Index(i).Child("visualization"),
						"must be set on a dataLayer",
					))
				}
				if wms.Spec.Service.Mapfile != nil && style.Visualization != nil {
					layerErrs = append(layerErrs, field.Invalid(
						path.Child("styles").Index(i).Child("visualization"),
						style.Visualization,
						"must not be set on a layer with a static mapfile",
					))
				}
			}
		}

		if layerType == GroupLayer || layerType == TopLayer {
			if !layer.Visible {
				layerErrs = append(layerErrs, field.Invalid(
					path.Child("visible"),
					layer.Visible,
					"must be true for a "+layerType,
				))
			}
			if layer.Data != nil {
				layerErrs = append(layerErrs, field.Invalid(
					path.Child("data"),
					"",
					"must not be set on a grouplayer",
				))
			}
			for i, style := range layer.Styles {
				if style.Visualization != nil {
					layerErrs = append(layerErrs, field.Invalid(
						path.Child("styles").Index(i),
						style.Visualization,
						"must not be set on a groupLayer",
					))
				}
			}
		}
		if len(layerErrs) != 0 {
			*allErrs = append(*allErrs, layerErrs...)
		}
	}

	if !hasVisibleLayer {
		*allErrs = append(*allErrs, field.Invalid(
			path.Child("layers"),
			"",
			"at least one layer must be visible",
		))
	}

	ValidateInspire(wms, allErrs)

	if wms.Spec.HorizontalPodAutoscalerPatch != nil {
		ValidateHorizontalPodAutoscalerPatch(*wms.Spec.HorizontalPodAutoscalerPatch, allErrs)
	}

	podSpecPatch := wms.Spec.PodSpecPatch
	ValidateEphemeralStorage(podSpecPatch, allErrs)
}

func findEqualChildStyleNames(layer *Layer, equalStyleNames *map[string][]string) {
	if len(layer.Layers) == 0 {
		return
	}
	equalChildStyleNames := map[string][]string{}
	for _, childLayer := range layer.Layers {
		if childLayer.Name == nil {
			// Name check is done elsewhere
			// To prevent errors here we just continue
			continue
		}

		var equalStyles []string
		for _, style := range layer.Styles {
			for _, childStyle := range childLayer.Styles {
				if style.Name == childStyle.Name {
					equalStyles = append(equalStyles, style.Name)
				}
			}
		}
		if len(equalStyles) > 0 {
			equalChildStyleNames[*childLayer.Name] = equalStyles
		}
		findEqualChildStyleNames(&childLayer, equalStyleNames)
	}
	maps.Copy(*equalStyleNames, equalChildStyleNames)
}
