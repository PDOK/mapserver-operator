package v3

import (
	"fmt"
	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"
	"k8s.io/utils/strings/slices"
	"maps"
	"strings"
)

func (wms *WMS) ValidateCreate() ([]string, error) {
	var warnings []string
	var reasons []string

	err := sharedValidation.ValidateLabelsOnCreate(wms.Labels)
	if err != nil {
		reasons = append(reasons, fmt.Sprintf("%v", err))
	}

	validateWMS(wms, &warnings, &reasons)

	if len(reasons) > 0 {
		return warnings, fmt.Errorf("%s", strings.Join(reasons, ". "))
	}

	return warnings, nil
}

func (wms *WMS) ValidateUpdate(wmsOld *WMS) ([]string, error) {
	var warnings []string
	var reasons []string

	// Check labels did not change
	err := sharedValidation.ValidateLabelsOnUpdate(wmsOld.Labels, wms.Labels)
	if err != nil {
		reasons = append(reasons, fmt.Sprintf("%v", err))
	}

	// Check service.baseURL did not change
	if wms.Spec.Service.URL != wmsOld.Spec.Service.URL {
		reasons = append(reasons, "service.baseURL is immutable")
	}

	if (wms.Spec.Service.Inspire == nil && wmsOld.Spec.Service.Inspire != nil) || (wms.Spec.Service.Inspire != nil && wmsOld.Spec.Service.Inspire == nil) {
		reasons = append(reasons, "services cannot change from inspire to not inspire or the other way around")
	}

	validateWMS(wms, &warnings, &reasons)

	if len(reasons) > 0 {
		return warnings, fmt.Errorf("%s", strings.Join(reasons, ". "))
	}

	return warnings, nil
}

func validateWMS(wms *WMS, warnings *[]string, reasons *[]string) {
	if strings.Contains(wms.GetName(), "wms") {
		*warnings = append(*warnings, sharedValidation.FormatValidationWarning("name should not contain wms", wms.GroupVersionKind(), wms.GetName()))
	}

	service := wms.Spec.Service

	err := sharedValidation.ValidateBaseURL(service.URL)
	if err != nil {
		*reasons = append(*reasons, fmt.Sprintf("%v", err))
	}

	if service.Mapfile != nil {
		if service.Resolution != nil {
			*warnings = append(*warnings, sharedValidation.FormatValidationWarning("service.resolution is not used when service.mapfile is configured", wms.GroupVersionKind(), wms.GetName()))
		}
		if service.DefResolution != nil {
			*warnings = append(*warnings, sharedValidation.FormatValidationWarning("service.defResolution is not used when service.mapfile is configured", wms.GroupVersionKind(), wms.GetName()))
		}
	}

	var rewriteGroupToDataLayers = false
	if wms.Spec.Options.RewriteGroupToDataLayers != nil {
		rewriteGroupToDataLayers = *wms.Spec.Options.RewriteGroupToDataLayers
	}

	var validateChildStyleNameEqual = true
	if wms.Spec.Options.ValidateChildStyleNameEqual != nil {
		validateChildStyleNameEqual = *wms.Spec.Options.ValidateChildStyleNameEqual
	}

	equalChildStyleNames := map[string][]string{}
	if rewriteGroupToDataLayers && validateChildStyleNameEqual {
		findEqualChildStyleNames(&wms.Spec.Service.Layer, &equalChildStyleNames)
	}

	var names []string
	hasVisibleLayer := false
	wms.Spec.Service.Layer.setInheritedBoundingBoxes()
	for _, layer := range wms.Spec.Service.Layer.GetAllLayers() {
		var layerReasons []string

		layerType := layer.GetLayerType(&service)
		var layerName string
		if layer.Name == nil {
			if layerType != TopLayer {
				layerReasons = append(layerReasons, "layer.Name is required (except for the toplayer)")
			}
			layerName = "unnamed:" + layerType
		} else {
			layerName = *layer.Name
		}

		if slices.Contains(names, layerName) {
			layerReasons = append(layerReasons, fmt.Sprintf("layer names must be unique, layer.name '%s' is duplicated", layerName))
		}
		names = append(names, layerName)

		if service.Mapfile != nil && layer.BoundingBoxes != nil {
			*warnings = append(*warnings, sharedValidation.FormatValidationWarning("layer.boundingBoxes is not used when service.mapfile is configured", wms.GroupVersionKind(), wms.GetName()))
		}
		if service.Mapfile == nil && service.DataEPSG != "EPSG:28992" && !layer.hasBoundingBoxForCRS(service.DataEPSG) {
			layerReasons = append(layerReasons, "layer.boundingBoxes must contain a boundingBox for CRS '"+service.DataEPSG+"' when service.dataEPSG is not 'EPSG:28992'")
		}

		if layer.Visible != nil && !*layer.Visible {
			if layer.Title != nil {
				*warnings = append(*warnings, sharedValidation.FormatValidationWarning("layer.title is not used when layer.visible=false", wms.GroupVersionKind(), wms.GetName()))
			}
			if layer.Abstract != nil {
				*warnings = append(*warnings, sharedValidation.FormatValidationWarning("layer.abstract is not used when layer.visible=false", wms.GroupVersionKind(), wms.GetName()))
			}
			if layer.Keywords != nil {
				*warnings = append(*warnings, sharedValidation.FormatValidationWarning("layer.keywords is not used when layer.visible=false", wms.GroupVersionKind(), wms.GetName()))
			}
			if layer.DatasetMetadataURL != nil {
				*warnings = append(*warnings, sharedValidation.FormatValidationWarning("layer.datasetMetadataURL is not used when layer.visible=false", wms.GroupVersionKind(), wms.GetName()))
			}
			if layer.Authority != nil && layer.Authority.SpatialDatasetIdentifier != "" {
				*warnings = append(*warnings, sharedValidation.FormatValidationWarning("layer.authority.spatialDatasetIdentifier is not used when layer.visible=false", wms.GroupVersionKind(), wms.GetName()))
			}
			for _, style := range layer.Styles {
				if style.Title != nil {
					*warnings = append(*warnings, sharedValidation.FormatValidationWarning("style.title is not used when layer.visible=false", wms.GroupVersionKind(), wms.GetName()))
				}
				if style.Abstract != nil {
					*warnings = append(*warnings, sharedValidation.FormatValidationWarning("style.abstract is not used when layer.visible=false", wms.GroupVersionKind(), wms.GetName()))
				}
			}
		}

		if layer.Visible != nil && *layer.Visible {
			var fields []string
			hasVisibleLayer = true

			if layer.Title == nil {
				fields = append(fields, "layer.title")
			}
			if layer.Abstract == nil {
				fields = append(fields, "layer.abstract")
			}
			if layer.Keywords == nil {
				fields = append(fields, "layer.keywords")
			}
			if len(fields) != 0 {
				layerReasons = append(layerReasons, "layer.visible=true; missing required fields: "+strings.Join(fields, ", "))
			}
			for _, style := range layer.Styles {
				if style.Title == nil {
					layerReasons = append(layerReasons, fmt.Sprintf("invalid style: '%s': style.title must be set on a visible layer", style.Name))
				}
			}
			if !rewriteGroupToDataLayers && validateChildStyleNameEqual {
				equalStylesNames, ok := equalChildStyleNames[layerName]
				if ok {
					for _, styleName := range equalStylesNames {
						layerReasons = append(layerReasons, fmt.Sprintf("invalid style: '%s': style.name from parent layer must not be set on a child layer", styleName))
					}
				}
			}
		}

		if layer.IsDataLayer() {
			for _, style := range layer.Styles {
				if wms.Spec.Service.Mapfile == nil && style.Visualization == nil {
					layerReasons = append(layerReasons, fmt.Sprintf("invalid style: '%s': style.visualization must be set on a dataLayer", style.Name))
				}
				if wms.Spec.Service.Mapfile != nil && style.Visualization != nil {
					layerReasons = append(layerReasons, fmt.Sprintf("invalid style: '%s': style.visualization must not be set on a layer when a static mapfile is used", style.Name))
				}
			}
		}

		if layerType == GroupLayer || layerType == TopLayer {
			if layer.Visible != nil && !*layer.Visible {
				layerReasons = append(layerReasons, layerType+" must be visible")
			}
			if layer.Data != nil {
				layerReasons = append(layerReasons, "layer.data must not be set on a groupLayer")
			}
			for _, style := range layer.Styles {
				if style.Visualization != nil {
					layerReasons = append(layerReasons, fmt.Sprintf("invalid style: '%s': style.visualization must not be set on a groupLayer", style.Name))
				}
			}
		}
		if len(layerReasons) != 0 {
			*reasons = append(*reasons, fmt.Sprintf("%s '%s' is invalid: ", layerType, layerName)+strings.Join(layerReasons, ", "))
		}
	}

	if !hasVisibleLayer {
		*reasons = append(*reasons, "at least one layer must be visible")
	}
}

func findEqualChildStyleNames(layer *Layer, equalStyleNames *map[string][]string) {
	if layer.Layers == nil || len(*layer.Layers) == 0 {
		return
	}
	equalChildStyleNames := map[string][]string{}
	for _, childLayer := range *layer.Layers {
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
