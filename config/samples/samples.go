package samples

import (
	_ "embed"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"sigs.k8s.io/yaml"
)

//go:embed v3_wfs.yaml
var v3WFSContent string

func V3WFS() (*pdoknlv3.WFS, error) {
	var sample pdoknlv3.WFS
	err := yaml.Unmarshal([]byte(v3WFSContent), &sample)
	return &sample, err
}
