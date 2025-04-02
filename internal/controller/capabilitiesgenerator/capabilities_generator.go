package capabilitiesgenerator

import (
	"errors"
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func GetInput[W pdoknlv3.WMSWFS](webservice W, ownerInfo *smoothoperatorv1.OwnerInfo) (input string, err error) {
	switch any(webservice).(type) {
	case *pdoknlv3.WFS:
		if WFS, ok := any(webservice).(*pdoknlv3.WFS); ok {
			return createInputForWFS(WFS, ownerInfo)
		}
	case *pdoknlv3.WMS:
		if _, ok := any(webservice).(*pdoknlv3.WMS); ok {
			return "", errors.New("not implemented for WMS")
		}
	default:
		return "", fmt.Errorf("unexpected input, webservice should be of type WFS or WMS, webservice: %v", webservice)
	}
	return "", fmt.Errorf("unexpected input, webservice should be of type WFS or WMS, webservice: %v", webservice)
}

func createInputForWFS(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	input, err := MapWFSToCapabilitiesGeneratorInput(wfs, ownerInfo)
	if err != nil {
		return "", err
	}
	yamlInput, err := yaml.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal the capabilities generator input to yaml: %w", err)
	}

	return string(yamlInput), nil
}
