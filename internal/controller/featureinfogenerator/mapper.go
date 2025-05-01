package featureinfogenerator

import (
	featureinfo "github.com/pdok/featureinfo-generator/pkg/types"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

const (
	featureinfoGeneratorSchemaVersion = 2
)

func MapWMSToFeatureinfoGeneratorInput(wms *pdoknlv3.WMS) (*featureinfo.Scheme, error) {

	input := &featureinfo.Scheme{
		AutomaticCasing: wms.Spec.Options.AutomaticCasing,
		Version:         featureinfoGeneratorSchemaVersion,
		Layers:          []featureinfo.Layer{},
	}

	for _, layer := range wms.Spec.Service.Layer.GetAllLayers() {
		if !layer.IsDataLayer() {
			continue
		}
		l := featureinfo.Layer{
			Name:       *layer.Name,
			Properties: getProperties(&layer),
		}

		parentLayer := layer.GetParent(&wms.Spec.Service.Layer)
		if parentLayer != nil && parentLayer.IsGroupLayer() {
			l.GroupName = smoothoperatorutils.PointerVal(parentLayer.Name, "")
		}

		input.Layers = append(input.Layers, l)
	}
	return input, nil
}

func getProperties(layer *pdoknlv3.Layer) (properties []featureinfo.Property) {
	switch {
	case layer.Data.Gpkg != nil:
		properties = getPropertiesForVector(layer.Data.Gpkg.Columns)
	case layer.Data.Postgis != nil:
		properties = getPropertiesForVector(layer.Data.Postgis.Columns)
	case layer.Data.TIF != nil:
		properties = getPropertiesForRaster(layer.Data.TIF.GetFeatureInfoIncludesClass)
	}
	return
}

func getPropertiesForVector(columns []pdoknlv3.Column) (properties []featureinfo.Property) {
	properties = append(properties, featureinfo.Property{Name: "fuuid"})
	for _, column := range columns {
		prop := featureinfo.Property{Name: column.Name}
		if column.Alias != nil {
			prop.Alias = *column.Alias
		}
		properties = append(properties, prop)
	}
	return
}

func getPropertiesForRaster(includeClass *bool) (properties []featureinfo.Property) {
	properties = append(properties, featureinfo.Property{Name: "value_list"})
	if includeClass != nil && *includeClass {
		properties = append(properties, featureinfo.Property{Name: "class"})
	}
	return
}
