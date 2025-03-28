// TODO Move this to the ogc-capabilities-generator on Github

package capabilitiesgenerator

import (
	"github.com/pdok/ogc-specifications/pkg/wcs201"
	"github.com/pdok/ogc-specifications/pkg/wfs200"
	"github.com/pdok/ogc-specifications/pkg/wms130"
	"github.com/pdok/ogc-specifications/pkg/wmts100"
)

// Config is the base struct for the config yaml
type Config struct {
	Global   Global   `yaml:"global"`
	Services Services `yaml:"services"`
}

// Global contains a collection of base var that are globally usable for the generation of the capabilities
type Global struct {
	Prefix                    string  `yaml:"prefix"`
	Namespace                 string  `yaml:"namespace"`
	OnlineResourceurl         string  `yaml:"onlineResourceUrl"`
	Path                      string  `yaml:"path"`
	Version                   string  `yaml:"version"`
	AdditionalSchemaLocations string  `yaml:"additionalSchemaLocations"`
	Empty                     *string `yaml:"-"`
}

// Services contain a single service struct for every service type
type Services struct {
	WFS200Config  *WFS200Config  `yaml:"wfs200,omitempty"`
	WMS130Config  *WMS130Config  `yaml:"wms130,omitempty"`
	WMTS100Config *WMTS100Config `yaml:"wmts100,omitempty"`
	WCS201Config  *WCS201Config  `yaml:"wcs201,omitempty"`
}

// The WFS200Config service struct
type WFS200Config struct {
	Filename string                         `yaml:"filename"`
	Wfs200   wfs200.GetCapabilitiesResponse `yaml:"definition"`
}

// The WMS130Config service struct
type WMS130Config struct {
	Filename string                         `yaml:"filename"`
	Wms130   wms130.GetCapabilitiesResponse `yaml:"definition"`
}

// The WMTS100Config service struct
type WMTS100Config struct {
	Filename string                          `yaml:"filename"`
	Wmts100  wmts100.GetCapabilitiesResponse `yaml:"definition"`
}

// The WCS201Config service struct
type WCS201Config struct {
	Filename string                         `yaml:"filename"`
	Wcs201   wcs201.GetCapabilitiesResponse `yaml:"definition"`
}
