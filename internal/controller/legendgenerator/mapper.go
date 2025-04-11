package legendgenerator

import (
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	defaultMapserverConf = `CONFIG
  ENV
    MS_MAP_NO_PATH "true"
  END
END
`
)

type LegendReference struct {
	Layer string `yaml:"layer" json:"layer"`
	Style string `yaml:"style" json:"style"`
}

type OgcWebserviceProxyConfig struct {
	GroupLayers map[string][]string `yaml:"grouplayers" json:"grouplayers"`
}

func getBareConfigMapLegendGenerator(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-legend-generator",
			Namespace: obj.GetNamespace(),
		},
	}
}

func GetLegendGeneratorConfigMap(wms *pdoknlv3.WMS) *corev1.ConfigMap {
	result := getBareConfigMapLegendGenerator(wms)
	labels := controller.AddCommonLabels(wms, smoothoperatorutils.CloneOrEmptyMap(wms.GetLabels()))
	result.Labels = labels

	result.Immutable = smoothoperatorutils.Pointer(true)
	result.Data = map[string]string{}
	result.Data["default_mapserver.conf"] = defaultMapserverConf

	addLayerInput(wms, result.Data)

	if wms.Spec.Options.RewriteGroupToDataLayers != nil && *wms.Spec.Options.RewriteGroupToDataLayers {
		addLegendFixerConfig(wms, result.Data)
	}

	return result
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
				Layer: layer.Name,
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
	fileBytes, err := os.ReadFile("./legend-fixer.sh")
	if err == nil {
		data["legend-fixer.sh"] = string(fileBytes)
	}

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
						Layer: layer.Name,
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

	if topLayer.IsGroupLayer() {
		layerName := topLayer.Name
		targetArray := make([]string, 0)
		getAllNestedNonGroupLayerNames(&topLayer, &targetArray)
		groupLayers[layerName] = targetArray

		for _, subLayer := range *topLayer.Layers {
			if subLayer.IsGroupLayer() {
				layerName = subLayer.Name
				targetArray = make([]string, 0)
				getAllNestedNonGroupLayerNames(&subLayer, &targetArray)
				groupLayers[layerName] = targetArray
			}
		}
	}

	ogcWebServiceProxyConfig := OgcWebserviceProxyConfig{GroupLayers: groupLayers}
	proxyConfigData, err := yaml.Marshal(ogcWebServiceProxyConfig)
	data["ogc-webservice-proxy-config.yaml"] = string(proxyConfigData)
}

func getAllNestedNonGroupLayerNames(layer *pdoknlv3.Layer, target *[]string) {
	for _, subLayer := range *layer.Layers {
		if subLayer.IsGroupLayer() {
			getAllNestedNonGroupLayerNames(&subLayer, target)
		} else {
			*target = append(*target, subLayer.Name)
		}
	}
}
