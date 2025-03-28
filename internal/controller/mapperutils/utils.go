package mapperutils

import (
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"strings"
)

func GetNamespaceURI(prefix string, ownerInfo *smoothoperatorv1.OwnerInfo) string {
	return strings.ReplaceAll(ownerInfo.Spec.NamespaceTemplate, "{{prefix}}", prefix)
}

func EscapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

func GetPath(WFS *pdoknlv3.WFS) (path string) {
	// TODO make this generic for WMS
	webserviceType := "wfs"
	datasetOwner := GetLabelValueByKey(WFS.ObjectMeta.Labels, "dataset-owner")
	dataset := GetLabelValueByKey(WFS.ObjectMeta.Labels, "dataset")
	theme := GetLabelValueByKey(WFS.ObjectMeta.Labels, "theme")
	serviceVersion := GetLabelValueByKey(WFS.ObjectMeta.Labels, "service-version")

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

func GetLabelValueByKey(labels map[string]string, key string) *string {
	for k, v := range labels {
		if k == key {
			return &v
		}
	}
	return nil
}
