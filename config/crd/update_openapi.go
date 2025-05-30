package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	goyaml "gopkg.in/yaml.v3"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kyaml "sigs.k8s.io/yaml"
)

// Usage: go run ./update_layersv3_openapi.go <crd_dir_path>
func main() {
	crdDir := os.Args[1]

	updateWMSV3(crdDir)
	updateWFSV3(crdDir)
}

func updateWMSV3(crdDir string) {
	path := filepath.Join(crdDir, "pdok.nl_wms.yaml")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(errors.Wrap(err, "WMS v3 manifest not found"))
	}

	content, _ := os.ReadFile(path)
	crd := &v1.CustomResourceDefinition{}
	err := kyaml.Unmarshal(content, &crd)
	if err != nil {
		panic(err)
	}

	versions := make([]v1.CustomResourceDefinitionVersion, 0)
	for _, version := range crd.Spec.Versions {
		if version.Name == "v3" {
			updateMapfileV3(&version)
			updateLayersV3(&version)

			versions = append(versions, version)
		} else {
			versions = append(versions, version)
		}
	}

	crd.Spec.Versions = versions
	updatedContent, _ := kyaml.Marshal(crd)

	// Remove the 'status' field from the yaml
	var rawData map[string]interface{}
	_ = goyaml.Unmarshal(updatedContent, &rawData)
	delete(rawData, "status")

	f, _ := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY, 0644)
	defer f.Close()

	enc := goyaml.NewEncoder(f)
	defer enc.Close()

	enc.SetIndent(2)
	_ = enc.Encode(rawData)
}

func updateLayersV3(version *v1.CustomResourceDefinitionVersion) {
	schema := version.Schema.OpenAPIV3Schema
	spec := schema.Properties["spec"]
	service := spec.Properties["service"]
	layer := service.Properties["layer"]

	// Level 3
	layerSpecLevel3 := layer.DeepCopy()
	layerSpecLevel3.Required = append(layerSpecLevel3.Required, "name")
	delete(layerSpecLevel3.Properties, "layers")
	xvals := v1.ValidationRules{}
	for _, xval := range layerSpecLevel3.XValidations {
		if !strings.Contains(xval.Rule, "self.layers") {
			xvals = append(xvals, xval)
		}
	}
	layerSpecLevel3.XValidations = xvals

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
}

func updateWFSV3(crdDir string) {
	path := filepath.Join(crdDir, "pdok.nl_wfs.yaml")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(errors.Wrap(err, "WFS v3 manifest not found"))
	}

	content, _ := os.ReadFile(path)
	crd := &v1.CustomResourceDefinition{}
	err := kyaml.Unmarshal(content, &crd)
	if err != nil {
		panic(err)
	}

	versions := make([]v1.CustomResourceDefinitionVersion, 0)
	for _, version := range crd.Spec.Versions {
		if version.Name == "v3" {
			updateMapfileV3(&version)

			versions = append(versions, version)
		} else {
			versions = append(versions, version)
		}
	}

	crd.Spec.Versions = versions
	updatedContent, _ := kyaml.Marshal(crd)

	// Remove the 'status' field from the yaml
	var rawData map[string]interface{}
	_ = goyaml.Unmarshal(updatedContent, &rawData)
	delete(rawData, "status")

	f, _ := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY, 0644)
	defer f.Close()

	enc := goyaml.NewEncoder(f)
	defer enc.Close()

	enc.SetIndent(2)
	_ = enc.Encode(rawData)
}

func updateMapfileV3(version *v1.CustomResourceDefinitionVersion) {
	schema := version.Schema.OpenAPIV3Schema
	spec := schema.Properties["spec"]
	service := spec.Properties["service"]
	mapfile := service.Properties["mapfile"]
	configMapKeyRef := mapfile.Properties["configMapKeyRef"]
	configMapKeyRef.Required = append(configMapKeyRef.Required, "name")
	name := configMapKeyRef.Properties["name"]
	name.Default = nil
	name.Description = "Name of the referent."

	configMapKeyRef.Properties["name"] = name
	mapfile.Properties["configMapKeyRef"] = configMapKeyRef
	service.Properties["mapfile"] = mapfile
	spec.Properties["service"] = service
	schema.Properties["spec"] = spec
	version.Schema = &v1.CustomResourceValidation{
		OpenAPIV3Schema: schema,
	}
}
