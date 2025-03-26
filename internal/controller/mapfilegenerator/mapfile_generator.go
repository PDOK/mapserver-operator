package mapfilegenerator

import (
	"encoding/json"
	"errors"
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
)

func GetConfig[W *pdoknlv3.WFS | *pdoknlv3.WMS](webservice W, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	switch any(webservice).(type) {
	case *pdoknlv3.WFS:
		if WFS, ok := any(webservice).(*pdoknlv3.WFS); ok {
			return createConfigForWFS(WFS, ownerInfo)
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

func createConfigForWFS(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	input, err := MapWFSToMapfileGeneratorInput(wfs, ownerInfo)
	if err != nil {
		return "", err
	}

	jsonConfig, err := json.MarshalIndent(input, "", "    ")
	if err != nil {
		return "", err
	}
	return string(jsonConfig), nil
}
