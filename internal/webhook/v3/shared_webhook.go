package v3

import (
	"encoding/json"
	"errors"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"os"
	"sigs.k8s.io/yaml"
)

const (
	samplesPath = "../../../config/samples/"
)

func getSampleFilename[W pdoknlv3.WMSWFS](webservice W) (string, error) {
	switch any(webservice).(type) {
	case *pdoknlv3.WFS:
		if _, ok := any(webservice).(*pdoknlv3.WFS); ok {
			return samplesPath + "v3_wfs.yaml", nil
		}
	case *pdoknlv3.WMS:
		if _, ok := any(webservice).(*pdoknlv3.WMS); ok {
			return samplesPath + "v3_wms.yaml", nil
		}
	}
	return "", errors.New("unknown webservice type, cannot determine sample filename")
}

func readSample[W pdoknlv3.WMSWFS](webservice W) error {
	sampleFilename, err := getSampleFilename(webservice)
	if err != nil {
		return err
	}
	sampleYaml, err := os.ReadFile(sampleFilename)
	if err != nil {
		return err
	}
	sampleJSON, err := yaml.YAMLToJSONStrict(sampleYaml)
	if err != nil {
		return err
	}
	err = json.Unmarshal(sampleJSON, webservice)
	if err != nil {
		return err
	}

	return nil
}
