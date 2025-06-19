package mapperutils

import (
	"strings"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	corev1 "k8s.io/api/core/v1"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func GetContainerResourceRequest[O pdoknlv3.WMSWFS](obj O, containerName string, resource corev1.ResourceName) *resource.Quantity {
	for _, container := range obj.PodSpecPatch().Containers {
		if container.Name == containerName {
			q := container.Resources.Requests[resource]
			if !q.IsZero() {
				return &q
			}
		}
	}

	return nil
}

func GetContainerResourceLimit[O pdoknlv3.WMSWFS](obj O, containerName string, resource corev1.ResourceName) *resource.Quantity {
	for _, container := range obj.PodSpecPatch().Containers {
		if container.Name == containerName {
			q := container.Resources.Limits[resource]
			if !q.IsZero() {
				return &q
			}
		}
	}

	return nil
}

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
	return GetContainerResourceLimit(obj, constants.MapserverName, corev1.ResourceEphemeralStorage)
}

func EphemeralStorageRequest[O pdoknlv3.WMSWFS](obj O) *resource.Quantity {
	return GetContainerResourceRequest(obj, constants.MapserverName, corev1.ResourceEphemeralStorage)
}

func GetNamespaceURI(prefix string, ownerInfo *smoothoperatorv1.OwnerInfo) string {
	return strings.ReplaceAll(*ownerInfo.Spec.NamespaceTemplate, "{{prefix}}", prefix)
}

func AnyMatch[S ~[]E, E any](slice S, eql func(E) bool) bool {
	for _, elem := range slice {
		if eql(elem) {
			return true
		}
	}
	return false
}
