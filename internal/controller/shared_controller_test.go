package controller

import (
	"context"
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ownerInfoResourceName = "pdok"
	namespace             = "default"
	testImageName1        = "test.test/image:test1"
	testImageName2        = "test.test/image:test2"
	testImageName3        = "test.test/image:test3"
	testImageName4        = "test.test/image:test4"
	testImageName5        = "test.test/image:test5"
	testImageName6        = "test.test/image:test6"
)

func getHashedConfigMapNameFromClient[O pdoknlv3.WMSWFS](ctx context.Context, obj O, volumeName string) (string, error) {
	deployment := &appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: getBareDeployment(obj).GetName()}, deployment)
	if err != nil {
		return "", err
	}

	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == volumeName && volume.ConfigMap != nil {
			return volume.ConfigMap.Name, nil
		}
	}
	return "", fmt.Errorf("configmap %s not found", volumeName)
}

func getExpectedObjects[O pdoknlv3.WMSWFS](ctx context.Context, obj O, includeBlobDownload bool) ([]client.Object, error) {
	bareObjects := getSharedBareObjects(obj)
	var objects []client.Object

	// Remove ConfigMaps as they have hashed names
	for _, object := range bareObjects {
		if _, ok := object.(*v1.ConfigMap); !ok {
			objects = append(objects, object)
		}
	}

	// Add all ConfigMaps with hashed names
	cm := getBareConfigMap(obj)
	hashedName, err := getHashedConfigMapNameFromClient(ctx, obj, mapserver.ConfigMapVolumeName)
	if err != nil {
		return objects, err
	}
	cm.Name = hashedName
	objects = append(objects, cm)

	cm = getBareConfigMapMapfileGenerator(obj)
	hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, mapserver.ConfigMapMapfileGeneratorVolumeName)
	if err != nil {
		return objects, err
	}
	cm.Name = hashedName
	objects = append(objects, cm)

	cm = getBareConfigMapCapabilitiesGenerator(obj)
	hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, mapserver.ConfigMapCapabilitiesGeneratorVolumeName)
	if err != nil {
		return objects, err
	}
	cm.Name = hashedName
	objects = append(objects, cm)

	if includeBlobDownload {
		cm = getBareConfigMapBlobDownload(obj)
		hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, mapserver.ConfigMapBlobDownloadVolumeName)
		if err != nil {
			return objects, err
		}
		cm.Name = hashedName
		objects = append(objects, cm)
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		cm = getBareConfigMapLegendGenerator(obj)
		hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, mapserver.ConfigMapLegendGeneratorVolumeName)
		if err != nil {
			return objects, err
		}
		cm.Name = hashedName
		objects = append(objects, cm)

		cm = getBareConfigMapFeatureinfoGenerator(obj)
		hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, mapserver.ConfigMapFeatureinfoGeneratorVolumeName)
		if err != nil {
			return objects, err
		}
		cm.Name = hashedName
		objects = append(objects, cm)

		if obj.Options().UseWebserviceProxy() {
			cm = getBareConfigMapOgcWebserviceProxy(obj)
			hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, mapserver.ConfigMapOgcWebserviceProxyVolumeName)
			if err != nil {
				return objects, err
			}
			cm.Name = hashedName
			objects = append(objects, cm)
		}
	}

	return objects, nil
}
