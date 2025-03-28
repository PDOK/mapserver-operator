package static_files

import (
	"embed"
	"fmt"
)

//go:embed files
var embeddedFiles embed.FS

func GetStaticFiles() map[string][]byte {
	result := map[string][]byte{}

	files, _ := embeddedFiles.ReadDir("files")
	for _, f := range files {
		content, _ := embeddedFiles.ReadFile(fmt.Sprintf("files/%s", f.Name()))
		result[f.Name()] = content
	}

	return result
}
