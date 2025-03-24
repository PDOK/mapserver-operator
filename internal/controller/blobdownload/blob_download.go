package blobdownload

import (
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"os"
	"regexp"
	"strings"
)

const (
	scriptPath = "./gpkg_download.sh"
	tifPath    = "/srv/data/tif"
	imagesPath = "/srv/data/images"
	fontsPath  = "/srv/data/config/fonts"
	legendPath = "/var/www/legend"
)

func GetScript() (config string, err error) {
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func GetArgs[W pdoknlv3.WFS | pdoknlv3.WMS](webservice W) (args string, err error) {
	var sb strings.Builder

	switch any(webservice).(type) {
	case pdoknlv3.WFS:
		if WFS, ok := any(webservice).(pdoknlv3.WFS); ok {
			createConfig(&sb)
			downloadGeopackage(&sb, *WFS.Spec.Options.PrefetchData)
			// In case of WFS no downloads are needed for TIFFs, styling assets and legends
		}
	case pdoknlv3.WMS:
		if WMS, ok := any(webservice).(pdoknlv3.WMS); ok {
			createConfig(&sb)
			downloadGeopackage(&sb, *WMS.Spec.Options.PrefetchData)
			if err = downloadTiffs(&sb, &WMS); err != nil {
				return "", err
			}
			if err = downloadStylingAssets(&sb, &WMS); err != nil {
				return "", err
			}
			if err = downloadLegends(&sb, &WMS); err != nil {
				return "", err
			}
		}
	default:
		return "", fmt.Errorf("unexpected input, webservice should be of type WFS or WMS, webservice: %v", webservice)
	}
	return sb.String(), nil
}

func createConfig(sb *strings.Builder) {
	writeLine(sb, "set -e;")
	writeLine(sb, "mkdir -p /srv/data/config/;")
	writeLine(sb, "rclone config create --non-interactive --obscure blobs azureblob endpoint $BLOBS_ENDPOINT account $BLOBS_ACCOUNT key $BLOBS_KEY use_emulator true;")
}

func downloadGeopackage(sb *strings.Builder, prefetchData bool) {
	if prefetchData {
		writeLine(sb, "bash /srv/scripts/gpkg_download.sh;")
	}
}

func downloadTiffs(sb *strings.Builder, WMS *pdoknlv3.WMS) error {
	if !*WMS.Spec.Options.PrefetchData {
		return nil
	}

	for _, blobKey := range WMS.GetUniqueTiffBlobKeys() {
		fileName, err := getFilenameFromBlobKey(blobKey)
		if err != nil {
			return err
		}
		writeLine(sb, "rclone copyto blobs:/%s  %s/%s || exit 1;", blobKey, tifPath, fileName)
	}
	return nil
}

func downloadStylingAssets(sb *strings.Builder, WMS *pdoknlv3.WMS) error {
	for _, blobKey := range WMS.Spec.Service.StylingAssets.BlobKeys {
		fileName, err := getFilenameFromBlobKey(blobKey)
		if err != nil {
			return err
		}
		path := imagesPath
		isTTF, _ := regexp.MatchString(".*\\.(ttf)$", fileName)
		if isTTF {
			path = fontsPath
		}
		writeLine(sb, "rclone copyto blobs:/%s %s/%s || exit 1;", blobKey, path, fileName)
		if isTTF {
			fileRoot, err := getRootFromFilename(fileName)
			if err != nil {
				return err
			}
			writeLine(sb, "echo %s %s >> %s/fonts.list;", fileRoot, fileName, fontsPath)
		}
	}
	writeLine(sb, "echo 'generated fonts.list:';")
	writeLine(sb, "cat %v/fonts.list;", fontsPath)
	return nil
}

func downloadLegends(sb *strings.Builder, WMS *pdoknlv3.WMS) error {
	for _, layer := range WMS.GetAllLayersWithLegend() {
		writeLine(sb, "mkdir -p %s/%s;", legendPath, layer.Name)
		for _, style := range layer.Styles {
			writeLine(sb, "rclone copyto blobs:/%s  %s/%s/%s.png || exit 1;", style.Legend.BlobKey, legendPath, layer.Name, style.Name)
			fileName, err := getFilenameFromBlobKey(style.Legend.BlobKey)
			if err != nil {
				return err
			}
			writeLine(sb, "Copied legend %s to %s/%s/%s.png;", fileName, legendPath, layer.Name, style.Name)
		}
	}
	writeLine(sb, "chown -R 999:999 %s", legendPath)
	return nil
}

func getFilenameFromBlobKey(blobKey string) (string, error) {
	index := strings.LastIndex(blobKey, "/")
	if index == -1 {
		return "", fmt.Errorf("could not determine filename from blobkey %s", blobKey)
	}
	return blobKey[index+1:], nil
}

func getRootFromFilename(fileName string) (string, error) {
	index := strings.LastIndex(fileName, ".")
	if index == -1 {
		return "", fmt.Errorf("could not determine root from filename %s", fileName)
	}
	return fileName[:index], nil
}

func writeLine(sb *strings.Builder, format string, a ...any) {
	sb.WriteString(fmt.Sprintf(format, a...) + "\n")
}
