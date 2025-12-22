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
	wms, err := GetWmsCRD()
	if err != nil {
		panic(err)
	}

	err = validation.AddValidator(wms)
	if err != nil {
		panic(err)
	}

	wfs, err := GetWfsCRD()
	if err != nil {
		panic(err)
	}

	err = validation.AddValidator(wfs)
	if err != nil {
		panic(err)
	}
}

func GetWmsCRD() (v1.CustomResourceDefinition, error) {
	crd := v1.CustomResourceDefinition{}
	err := yaml.Unmarshal(wmsCRD, &crd)

	return crd, err
}

func GetWfsCRD() (v1.CustomResourceDefinition, error) {
	crd := v1.CustomResourceDefinition{}
	err := yaml.Unmarshal(wfsCRD, &crd)

	return crd, err
}
