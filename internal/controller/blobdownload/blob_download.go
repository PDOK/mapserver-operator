package blobdownload

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	"github.com/pdok/mapserver-operator/internal/controller/types"

	"k8s.io/utils/strings/slices"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	"github.com/pdok/mapserver-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	tifPath    = "/srv/data/tif"
	imagesPath = "/srv/data/images"
	fontsPath  = "/srv/data/config/fonts"
	legendPath = "/var/www/legend"
)

//go:embed gpkg_download.sh
var GpkgDownloadScript string

func GetScript() string {
	return GpkgDownloadScript
}

func GetBlobDownloadInitContainer[O pdoknlv3.WMSWFS](obj O, images types.Images, blobsConfigName, blobsSecretName string) (*corev1.Container, error) {
	blobkeys := []string{}
	for _, gpkg := range obj.GeoPackages() {
		// Deduplicate blobkeys to prevent double downloads
		if !slices.Contains(blobkeys, gpkg.BlobKey) {
			blobkeys = append(blobkeys, gpkg.BlobKey)
		}
	}

	initContainer := corev1.Container{
		Name:            constants.BlobDownloadName,
		Image:           images.MultitoolImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name:  "GEOPACKAGE_TARGET_PATH",
				Value: "/srv/data/gpkg",
			},
			{
				Name:  "GEOPACKAGE_DOWNLOAD_LIST",
				Value: strings.Join(blobkeys, ";"),
			},
		},
		EnvFrom: []corev1.EnvFromSource{
			utils.NewEnvFromSource(utils.EnvFromSourceTypeConfigMap, blobsConfigName),
			utils.NewEnvFromSource(utils.EnvFromSourceTypeSecret, blobsSecretName),
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("0.15"),
			},
		},
		Command: []string{"/bin/sh", "-c"},
		VolumeMounts: []corev1.VolumeMount{
			utils.GetBaseVolumeMount(),
			utils.GetDataVolumeMount(),
		},
	}

	// Additional blob-download configuration
	args, err := GetArgs(obj)
	if err != nil {
		return nil, err
	}
	initContainer.Args = []string{args}

	resourceCPU := resource.MustParse("0.2")
	if use, _ := mapperutils.UseEphemeralVolume(obj); use {
		resourceCPU = resource.MustParse("1")
	}
	initContainer.Resources.Limits = corev1.ResourceList{
		corev1.ResourceCPU: resourceCPU,
	}

	if obj.Options().PrefetchData {
		mount := corev1.VolumeMount{Name: constants.InitScriptsName, MountPath: "/srv/scripts", ReadOnly: true}
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, mount)
	}

	return &initContainer, nil
}

func GetArgs[W pdoknlv3.WMSWFS](webservice W) (args string, err error) {
	var sb strings.Builder

	switch any(webservice).(type) {
	case *pdoknlv3.WFS:
		if WFS, ok := any(webservice).(*pdoknlv3.WFS); ok {
			createConfig(&sb)
			downloadGeopackage(&sb, WFS.Options().PrefetchData)
			// In case of WFS no downloads are needed for TIFFs, styling assets and legends
		}
	case *pdoknlv3.WMS:
		if WMS, ok := any(webservice).(*pdoknlv3.WMS); ok {
			createConfig(&sb)
			downloadGeopackage(&sb, WMS.Options().PrefetchData)
			if err = downloadTiffs(&sb, WMS); err != nil {
				return "", err
			}
			if err = downloadStylingAssets(&sb, WMS); err != nil {
				return "", err
			}
			if err = downloadLegends(&sb, WMS); err != nil {
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

func downloadTiffs(sb *strings.Builder, wms *pdoknlv3.WMS) error {
	if !wms.Options().PrefetchData {
		return nil
	}

	for _, blobKey := range wms.GetUniqueTiffBlobKeys() {
		fileName, err := getFilenameFromBlobKey(blobKey)
		if err != nil {
			return err
		}
		writeLine(sb, "rclone copyto blobs:/%s %s/%s || exit 1;", blobKey, tifPath, fileName)
	}
	return nil
}

func downloadStylingAssets(sb *strings.Builder, wms *pdoknlv3.WMS) error {
	if wms.Spec.Service.StylingAssets == nil { // TODO Is StylingAssets required and should this return an error?
		return nil
	}

	generatedFontsList := false
	re := regexp.MustCompile(`.*\.(ttf)$`)
	for _, blobKey := range wms.Spec.Service.StylingAssets.BlobKeys {
		fileName, err := getFilenameFromBlobKey(blobKey)
		if err != nil {
			return err
		}
		path := imagesPath
		isTTF := re.MatchString(fileName)
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
			generatedFontsList = true
		}
	}

	if generatedFontsList {
		writeLine(sb, "echo 'generated fonts.list:';")
		writeLine(sb, "cat %v/fonts.list;", fontsPath)
	}

	return nil
}

func downloadLegends(sb *strings.Builder, wms *pdoknlv3.WMS) error {
	layers := wms.GetAllLayersWithLegend()
	if len(layers) > 0 {
		for _, layer := range layers {
			writeLine(sb, "mkdir -p %s/%s;", legendPath, *layer.Name)
			for _, style := range layer.Styles {
				writeLine(sb, "rclone copyto blobs:/%s %s/%s/%s.png || exit 1;", style.Legend.BlobKey, legendPath, *layer.Name, style.Name)
				fileName, err := getFilenameFromBlobKey(style.Legend.BlobKey)
				if err != nil {
					return err
				}
				writeLine(sb, "echo 'Copied legend %s to %s/%s/%s.png';", fileName, legendPath, *layer.Name, style.Name)
			}
		}
		writeLine(sb, "chown -R 999:999 %s", legendPath)
	}

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
