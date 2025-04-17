package main

import (
	"fmt"
	"github.com/pdok/mapserver-operator/api/v2beta1"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

func main() {
	var k8sClusters string
	fmt.Print("Enter k8s-cluster folder:\n")
	fmt.Scanln(&k8sClusters)
	if !strings.HasSuffix(k8sClusters, "/") {
		k8sClusters += "/"
	}
	k8sClusters += "applications"
	err := filepath.WalkDir(k8sClusters, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, "wms.yaml") {
			checkWms(path)
		} else if strings.HasSuffix(path, "wfs.yaml") {
			checkWfs(path)
		} else if strings.HasSuffix(path, "atom.yaml") {
			checkAtom(path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("impossible to walk directories: %s", err)
	}
}

func checkWms(path string) {
	print("Checking ")
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Could not read file '%s', exiting", path)
	}
	fileString := string(fileBytes)
	fileString = strings.ReplaceAll(fileString, "${BLOBS_RESOURCES_BUCKET}", "resources")
	fileString = strings.ReplaceAll(fileString, "${OWNER}", "owner")
	fileString = strings.ReplaceAll(fileString, "${DATASET}", "dataset")
	fileString = strings.ReplaceAll(fileString, "${SERVICE_VERSION}", "v1_0")
	fileString = strings.ReplaceAll(fileString, "${THEME}", "theme")
	fileString = strings.ReplaceAll(fileString, "${INCLUDES}", "includes")
	fileString = strings.ReplaceAll(fileString, "${BLOBS_GEOPACKAGES_BUCKET}", "geopackages")
	fileString = strings.ReplaceAll(fileString, "${BLOBS_TIFS_BUCKET}", "tifs")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION}", "bbbb/2")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_25}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_50}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_100}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_250}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_500}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_1000}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_1}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${BLOBS_DOWNLOADS_BUCKET}", "downloads")
	fileString = strings.ReplaceAll(fileString, "${LIMITS_EPHEMERAL_STORAGE}", "102M")
	fileString = strings.ReplaceAll(fileString, "${REQUESTS_CPU}", "1001")
	fileString = strings.ReplaceAll(fileString, "${REQUESTS_MEM}", "100M")
	fileString = strings.ReplaceAll(fileString, "${REQUESTS_EPHEMERAL_STORAGE}", "101M")

	var v2wms v2beta1.WMS
	err = yaml.Unmarshal([]byte(fileString), &v2wms)
	if err != nil {
		println(err)
		println(path)
		os.Exit(1)
	}
	var wms pdoknlv3.WMS
	v2beta1.V3HubFromV2(&v2wms, &wms)
}

func checkWfs(path string) {
	//print("Did not check ")
	//println(path)
}

func checkAtom(path string) {
	//print("Did not check ")
	//println(path)
}
