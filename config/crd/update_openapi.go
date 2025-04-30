package main

import (
	"github.com/pkg/errors"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

// Usage: go run ./update_layersv3_openapi.go <crd_dir_path>
func main() {
	crdDir := os.Args[1]

	updateWMSV3Layers(crdDir)
}

func updateWMSV3Layers(crdDir string) {
	path := filepath.Join(crdDir, "pdok.nl_wms.yaml")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(errors.Wrap(err, "WMS v3 manifest not found"))
	}

	content, _ := os.ReadFile(path)
	crd := &v1.CustomResourceDefinition{}
	err := yaml.Unmarshal(content, &crd)
	if err != nil {
		panic(err)
	}

	versions := make([]v1.CustomResourceDefinitionVersion, 0)
	for _, version := range crd.Spec.Versions {
		if version.Name == "v3" {
			schema := version.Schema.OpenAPIV3Schema
			spec := schema.Properties["spec"]
			service := spec.Properties["service"]
			layer := service.Properties["layer"]

			// Level 3
			layerSpecLevel3 := layer.DeepCopy()
			layerSpecLevel3.Required = append(layerSpecLevel3.Required, "name")
			delete(layerSpecLevel3.Properties, "layers")

			// Level 2
			layerSpecLevel2 := layer.DeepCopy()
			layerSpecLevel2.Required = append(layerSpecLevel2.Required, "name")
			layerSpecLevel2.Properties["layers"] = v1.JSONSchemaProps{
				Type:        "array",
				Description: "[OpenAPI spec injected by mapserver-operator/cmd/update_openapi.go]",
				Items:       &v1.JSONSchemaPropsOrArray{Schema: layerSpecLevel3},
			}

			layer.Properties["layers"] = v1.JSONSchemaProps{
				Type:        "array",
				Description: "[OpenAPI spec injected by mapserver-operator/cmd/update_openapi.go]",
				Items:       &v1.JSONSchemaPropsOrArray{Schema: layerSpecLevel2},
			}

			service.Properties["layer"] = layer
			spec.Properties["service"] = service
			schema.Properties["spec"] = spec
			version.Schema = &v1.CustomResourceValidation{
				OpenAPIV3Schema: schema,
			}

			versions = append(versions, version)
		} else {
			versions = append(versions, version)
		}
	}

	crd.Spec.Versions = versions
	updatedContent, _ := yaml.Marshal(crd)
	os.WriteFile(path, updatedContent, os.ModePerm)
}
