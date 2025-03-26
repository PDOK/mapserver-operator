package mapfilegenerator

import (
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	"strconv"
	"strings"
)

const (
	defaultMaxFeatures = 1000
	geopackagePath     = "/srv/data/gpkg"
)

func MapWFSToMapfileGeneratorInput(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (Input, error) {
	input := Input{
		Title:             escapeQuotes(wfs.Spec.Service.Title),
		Abstract:          escapeQuotes(wfs.Spec.Service.Abstract),
		Keywords:          strings.Join(wfs.Spec.Service.Keywords, ","),
		AccessConstraints: wfs.Spec.Service.AccessConstraints,
		Extent:            wfs.Spec.Service.Bbox.DefaultCRS.ToExtent(),
		WFSMaxFeatures:    getMaxFeatures(wfs.Spec.Service.CountDefault),
		NamespacePrefix:   wfs.Spec.Service.Prefix,
		NamespaceURI:      getNamespaceURI(wfs.Spec.Service.Prefix, ownerInfo),
		OnlineResource:    pdoknlv3.GetBaseURL(),
		Path:              getPath(wfs),
		MetadataId:        wfs.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier,
		AutomaticCasing:   wfs.Spec.Options.AutomaticCasing,
		DataEPSG:          wfs.Spec.Service.DefaultCrs,
		EPSGList:          wfs.Spec.Service.OtherCrs,
		Layers:            getLayers(wfs.Spec.Service.FeatureTypes),
	}

	return input, nil
}

func getMaxFeatures(countDefault *string) string {
	if countDefault != nil {
		return *countDefault
	}
	return strconv.Itoa(defaultMaxFeatures)
}

func getLabelValueByKey(labels map[string]string, key string) *string {
	for k, v := range labels {
		if k == key {
			return &v
		}
	}
	return nil
}

func getNamespaceURI(prefix string, ownerInfo *smoothoperatorv1.OwnerInfo) string {
	return strings.ReplaceAll(ownerInfo.Spec.NamespaceTemplate, "{{prefix}}", prefix)
}

func getPath(WFS *pdoknlv3.WFS) (path string) {
	// TODO make this generic for WMS
	webserviceType := "wfs"
	datasetOwner := getLabelValueByKey(WFS.ObjectMeta.Labels, "dataset-owner")
	dataset := getLabelValueByKey(WFS.ObjectMeta.Labels, "dataset")
	theme := getLabelValueByKey(WFS.ObjectMeta.Labels, "theme")
	serviceVersion := getLabelValueByKey(WFS.ObjectMeta.Labels, "service-version")

	path = fmt.Sprintf("/%s/%s", *datasetOwner, *dataset)
	if theme != nil {
		path += "/" + *theme
	}
	path += "/" + webserviceType
	if serviceVersion != nil {
		path += "/" + *serviceVersion
	}

	return path
}

func getLayers(featureTypes []pdoknlv3.FeatureType) (layers []Layer) {
	for _, featureType := range featureTypes {
		layer := Layer{
			Name:           featureType.Name,
			Title:          escapeQuotes(featureType.Title),
			Abstract:       escapeQuotes(featureType.Abstract),
			Keywords:       strings.Join(featureType.Keywords, ","),
			Extent:         featureType.Bbox.DefaultCRS.ToExtent(),
			MetadataId:     featureType.DatasetMetadataURL.CSW.MetadataIdentifier,
			Columns:        getColumns(featureType),
			TableName:      featureType.Data.GetTableName(),
			GeometryType:   featureType.Data.GetGeometryType(),
			GeopackagePath: getGeopackagePath(featureType),
		}
		if featureType.Data.Postgis != nil {
			layer.Postgis = smoothoperatorutils.Pointer(true)
		}

		layers = append(layers, layer)
	}

	return
}

func getColumns(featureType pdoknlv3.FeatureType) []Column {
	columns := []Column{{Name: "fuuid"}}
	for _, column := range *featureType.Data.GetColumns() {
		columns = append(columns, Column{Name: column.Name, Alias: column.Alias})
	}
	return columns
}

func getGeopackagePath(featureType pdoknlv3.FeatureType) *string {
	if featureType.Data.Gpkg == nil {
		return nil
	}
	index := strings.LastIndex(featureType.Data.Gpkg.BlobKey, "/") + 1
	blobName := featureType.Data.Gpkg.BlobKey[index:]
	return smoothoperatorutils.Pointer(geopackagePath + "/" + blobName)
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
