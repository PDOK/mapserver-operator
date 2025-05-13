package static

import (
	"embed"
	"sort"
)

//go:embed files
var embeddedFiles embed.FS

func GetStaticFiles() ([]string, map[string][]byte) {
	orderedNames := []string{}
	result := map[string][]byte{}

	files, _ := embeddedFiles.ReadDir("files")
	for _, f := range files {
		content, _ := embeddedFiles.ReadFile("files/" + f.Name())
		result[f.Name()] = content
		orderedNames = append(orderedNames, f.Name())
	}
	sort.Strings(orderedNames)

	return orderedNames, result
}
