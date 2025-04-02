package mapperutils

import (
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strings"
)

// Use ephemeral volume when ephemeral storage is greater then 10Gi
func UseEphemeralVolume[O pdoknlv3.WMSWFS](obj O) (bool, *resource.Quantity) {
	value := EphemeralStorageLimit(obj)
	threshold := resource.MustParse("10Gi")

	if value != nil {
		return value.Value() > threshold.Value(), value
	}

	return false, nil
}

func EphemeralStorageLimit[O pdoknlv3.WMSWFS](obj O) *resource.Quantity {
	return ephemeralStorage(obj, true)
}

func EphemeralStorageRequest[O pdoknlv3.WMSWFS](obj O) *resource.Quantity {
	return ephemeralStorage(obj, false)
}

func ephemeralStorage[O pdoknlv3.WMSWFS](obj O, limit bool) *resource.Quantity {
	for _, container := range obj.PodSpecPatch().Containers {
		if container.Name == "mapserver" {
			if limit {
				return container.Resources.Limits.StorageEphemeral()
			}

			return container.Resources.Requests.StorageEphemeral()
		}
	}

	return nil
}

func GetNamespaceURI(prefix string, ownerInfo *smoothoperatorv1.OwnerInfo) string {
	return strings.ReplaceAll(ownerInfo.Spec.NamespaceTemplate, "{{prefix}}", prefix)
}

func EscapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

func GetPath[O pdoknlv3.WMSWFS](obj O) (path string) {
	webserviceType := strings.ToLower(string(obj.Type()))
	datasetOwner := GetLabelValueByKey(obj.GetLabels(), "dataset-owner")
	dataset := GetLabelValueByKey(obj.GetLabels(), "dataset")
	theme := GetLabelValueByKey(obj.GetLabels(), "theme")
	serviceVersion := GetLabelValueByKey(obj.GetLabels(), "service-version")

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
