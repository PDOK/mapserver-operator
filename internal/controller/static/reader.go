package static

import (
	"embed"
	"slices"
)

//go:embed files
var embeddedFiles embed.FS

func GetStaticFiles() ([]string, map[string][]byte) {
	// Hardcoded order to get the same order as the old ansible operator
	orderedNames := []string{"include.conf", "ogc.lua", "default_mapserver.conf", "scraping-error.xml"}
	result := map[string][]byte{}

	files, _ := embeddedFiles.ReadDir("files")
	for _, f := range files {
		content, _ := embeddedFiles.ReadFile("files/" + f.Name())
		result[f.Name()] = content
		if !slices.Contains(orderedNames, f.Name()) {
			orderedNames = append(orderedNames, f.Name())
		}
	}

	return orderedNames, result
}
