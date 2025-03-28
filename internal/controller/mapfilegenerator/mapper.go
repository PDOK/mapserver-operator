package mapfilegenerator

import (
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"strconv"
	"strings"
)

const (
	defaultMaxFeatures = 1000
)

func MapWFSToMapfileGeneratorInput(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (Input, error) {
	input := Input{
		Title:             wfs.Spec.Service.Title,
		Abstract:          wfs.Spec.Service.Abstract,
		Keywords:          strings.Join(wfs.Spec.Service.Keywords, ","),
		AccessConstraints: wfs.Spec.Service.AccessConstraints,
		Extent:            wfs.Spec.Service.Bbox.DefaultCRS.ToExtent(),
		WFSMaxFeatures:    getMaxFeatures(wfs.Spec.Service.CountDefault),
		NamespacePrefix:   wfs.Spec.Service.Prefix,
		NamespaceURI:      getNamespaceURI(wfs.Spec.Service.Prefix, ownerInfo),
		OnlineResource:    pdoknlv3.GetHost(),
		Path:              getPath(wfs),
		MetadataId:        wfs.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier,
		DatasetOwner:      "", // Todo
		AuthorityURL:      "", // Todo
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
			Name:     featureType.Name,
			Title:    featureType.Name,
			Abstract: featureType.Abstract,
			// Todo expand
		}
		layers = append(layers, layer)
	}

	return
}
