package bases

import (
	_ "embed"
	"github.com/pdok/smooth-operator/pkg/validation"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

//go:embed pdok.nl_wfs.yaml
var wfsCRD []byte

//go:embed pdok.nl_wms.yaml
var wmsCRD []byte

func init() {
	wms := v1.CustomResourceDefinition{}
	err := yaml.Unmarshal(wmsCRD, &wms)
	if err != nil {
		panic(err)
	}

	err = validation.AddValidator(wms)
	if err != nil {
		panic(err)
	}

	wfs := v1.CustomResourceDefinition{}
	err = yaml.Unmarshal(wfsCRD, &wfs)
	if err != nil {
		panic(err)
	}

	err = validation.AddValidator(wfs)
	if err != nil {
		panic(err)
	}
}
