package legendgenerator

import (
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"

	_ "embed"
	"sigs.k8s.io/yaml"
	"strings"
)

// TODO Reuse default_mapserver.conf from static_files?
const (
	defaultMapserverConf = `CONFIG
  ENV
    MS_MAP_NO_PATH "true"
  END
END
`
)

//go:embed legend-fixer.sh
var legendFixerScript string

type LegendReference struct {
	Layer string `yaml:"layer" json:"layer"`
	Style string `yaml:"style" json:"style"`
}

type OgcWebserviceProxyConfig struct {
	GroupLayers map[string][]string `yaml:"grouplayers" json:"grouplayers"`
}

func addLayerInput(wms *pdoknlv3.WMS, data map[string]string) {
	legendReferences := make([]LegendReference, 0)

	if wms.Spec.Service.Layer.Layers != nil {
		for _, layer := range *wms.Spec.Service.Layer.Layers {
			processLayer(&layer, &legendReferences)
		}
	}

	sb := strings.Builder{}
	for _, reference := range legendReferences {
		sb.WriteString(fmt.Sprintf("\"%s\" \"%s\"\n", reference.Layer, reference.Style))
	}

	data["input"] = sb.String()
	referencesYaml, err := yaml.Marshal(legendReferences)
	if err == nil {
		data["input2"] = string(referencesYaml)
	}

}

func processLayer(layer *pdoknlv3.Layer, legendReferences *[]LegendReference) {
	if layer.Visible == nil || !*layer.Visible {
		return
	}
	for _, style := range layer.Styles {
		if style.Legend == nil {
			*legendReferences = append(*legendReferences, LegendReference{
				Layer: *layer.Name,
				Style: style.Name,
			})
		}
	}

	if layer.Layers != nil {
		for _, innerLayer := range *layer.Layers {
			processLayer(&innerLayer, legendReferences)
		}
	}
}

func addLegendFixerConfig(wms *pdoknlv3.WMS, data map[string]string) {
	data["legend-fixer.sh"] = legendFixerScript

	topLayer := wms.Spec.Service.Layer

	legendReferences := make([]LegendReference, 0)
	topLevelStyleNames := make(map[string]bool)

	for _, style := range topLayer.Styles {
		topLevelStyleNames[style.Name] = true
	}

	if topLayer.Layers != nil {
		// These layers are called 'middle layers' in the old operator
		for _, layer := range *wms.Spec.Service.Layer.Layers {
			for _, style := range layer.Styles {
				if topLevelStyleNames[style.Name] && style.Legend == nil {
					legendReferences = append(legendReferences, LegendReference{
						Layer: *layer.Name,
						Style: style.Name,
					})
				}
			}
		}
	}

	sb := strings.Builder{}
	for _, reference := range legendReferences {
		sb.WriteString(fmt.Sprintf("\"%s\" \"%s\"\n", reference.Layer, reference.Style))
	}

	data["remove"] = sb.String()

	groupLayers := make(map[string][]string)

	if topLayer.IsGroupLayer() && topLayer.Name != nil {
		layerName := topLayer.Name
		targetArray := make([]string, 0)
		getAllNestedNonGroupLayerNames(&topLayer, &targetArray)
		groupLayers[*layerName] = targetArray

		for _, subLayer := range *topLayer.Layers {
			if subLayer.IsGroupLayer() {
				layerName = subLayer.Name
				targetArray = make([]string, 0)
				getAllNestedNonGroupLayerNames(&subLayer, &targetArray)
				groupLayers[*layerName] = targetArray
			}
		}
	}

	ogcWebServiceProxyConfig := OgcWebserviceProxyConfig{GroupLayers: groupLayers}
	proxyConfigData, _ := yaml.Marshal(ogcWebServiceProxyConfig)
	data["ogc-webservice-proxy-config.yaml"] = string(proxyConfigData)
}

func getAllNestedNonGroupLayerNames(layer *pdoknlv3.Layer, target *[]string) {
	for _, subLayer := range *layer.Layers {
		if subLayer.IsGroupLayer() {
			getAllNestedNonGroupLayerNames(&subLayer, target)
		} else {
			*target = append(*target, *subLayer.Name)
		}
	}
}
