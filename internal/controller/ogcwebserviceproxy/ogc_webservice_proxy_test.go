package ogcwebserviceproxy

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

func TestGetConfig(t *testing.T) {
	tests := []string{"named-toplayer", "unnamed-toplayer"}

	for _, tt := range tests {
		input, err := os.ReadFile("test_data/input/" + tt + ".yaml")
		if err != nil {
			t.Errorf("os.ReadFile() error = %v", err)
		}
		wms := &pdoknlv3.WMS{}
		if err := yaml.Unmarshal(input, wms); err != nil {
			t.Errorf("yaml.Unmarshal() error = %v", err)
		}

		generated, err := MapWMSToOgcWebserviceProxyConfig(wms)
		if err != nil {
			t.Errorf("MapWMSToOgcWebserviceProxyConfig() error = %v", err)
		}

		expectedBytes, err := os.ReadFile("test_data/expected/" + tt + ".yaml")
		if err != nil {
			t.Errorf("os.ReadFile() error = %v", err)
		}

		var expected Config
		if err := yaml.Unmarshal(expectedBytes, &expected); err != nil {
			t.Errorf("yaml.Unmarshal() error = %v", err)
		}

		diff := cmp.Diff(expected, generated)
		if diff != "" {
			t.Errorf("GetConfig() mismatch (-want +got):\n%s", diff)
		}
	}
}
