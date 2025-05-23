package controller

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	"github.com/google/go-cmp/cmp"
	"github.com/pdok/mapserver-operator/api/v2beta1"
	"github.com/pdok/mapserver-operator/internal/controller/types"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorvalidation "github.com/pdok/smooth-operator/pkg/validation"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	v2 "k8s.io/api/autoscaling/v2"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace      = "default"
	testImageName1 = "test.test/image:test1"
	testImageName2 = "test.test/image:test2"
	testImageName3 = "test.test/image:test3"
	testImageName4 = "test.test/image:test4"
	testImageName5 = "test.test/image:test5"
	testImageName6 = "test.test/image:test6"
	testImageName7 = "test.test/image:test7"
)

func getHashedConfigMapNameFromClient[O pdoknlv3.WMSWFS](ctx context.Context, obj O, volumeName string) (string, error) {
	deployment := &appsv1.Deployment{}
	err := k8sClient.Get(ctx, k8stypes.NamespacedName{Namespace: obj.GetNamespace(), Name: getBareDeployment(obj).GetName()}, deployment)
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

func getExpectedObjects[O pdoknlv3.WMSWFS](ctx context.Context, obj O, includeBlobDownload bool, includeMapfileGeneratorConfigMap bool) ([]client.Object, error) {
	bareObjects := getSharedBareObjects(obj)
	var objects []client.Object

	// Remove ConfigMaps as they have hashed names
	for _, object := range bareObjects {
		if _, ok := object.(*corev1.ConfigMap); !ok {
			objects = append(objects, object)
		}
	}

	// Add all ConfigMaps with hashed names
	cm := getBareConfigMap(obj, constants.MapserverName)
	hashedName, err := getHashedConfigMapNameFromClient(ctx, obj, constants.MapserverName)
	if err != nil {
		return objects, err
	}
	cm.Name = hashedName
	objects = append(objects, cm)

	if includeMapfileGeneratorConfigMap {
		cm = getBareConfigMap(obj, constants.MapfileGeneratorName)
		hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, constants.ConfigMapMapfileGeneratorVolumeName)
		if err != nil {
			return objects, err
		}
		cm.Name = hashedName
		objects = append(objects, cm)
	}

	cm = getBareConfigMap(obj, constants.CapabilitiesGeneratorName)
	hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, constants.ConfigMapCapabilitiesGeneratorVolumeName)
	if err != nil {
		return objects, err
	}
	cm.Name = hashedName
	objects = append(objects, cm)

	if includeBlobDownload {
		cm = getBareConfigMap(obj, constants.InitScriptsName)
		hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, constants.InitScriptsName)
		if err != nil {
			return objects, err
		}
		cm.Name = hashedName
		objects = append(objects, cm)
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		wms, _ := any(obj).(*pdoknlv3.WMS)
		cm = getBareConfigMap(wms, constants.LegendGeneratorName)
		hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, constants.ConfigMapLegendGeneratorVolumeName)
		if err != nil {
			return objects, err
		}
		cm.Name = hashedName
		objects = append(objects, cm)

		cm = getBareConfigMap(wms, constants.FeatureinfoGeneratorName)
		hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, constants.ConfigMapFeatureinfoGeneratorVolumeName)
		if err != nil {
			return objects, err
		}
		cm.Name = hashedName
		objects = append(objects, cm)

		if obj.Options().UseWebserviceProxy() {
			cm = getBareConfigMap(wms, constants.OgcWebserviceProxyName)
			hashedName, err = getHashedConfigMapNameFromClient(ctx, obj, constants.ConfigMapOgcWebserviceProxyVolumeName)
			if err != nil {
				return objects, err
			}
			cm.Name = hashedName
			objects = append(objects, cm)
		}
	}

	return objects, nil
}

func testPath(t pdoknlv3.ServiceType, test string) string {
	return fmt.Sprintf("test_data/%s/%s/", strings.ToLower(string(t)), test)
}

func testMutate[T any](kind string, result *T, expectedFile string, mutate func(*T) error) {
	By("Testing mutating the " + kind)
	err := mutate(result)
	Expect(err).NotTo(HaveOccurred())

	var expected T
	data, err := os.ReadFile(expectedFile)
	Expect(err).NotTo(HaveOccurred())
	err = yaml.UnmarshalStrict(data, &expected)
	Expect(err).NotTo(HaveOccurred())

	diff := cmp.Diff(expected, *result)
	if diff != "" {
		Fail(diff)
	}

	By(fmt.Sprintf("Testing mutating the %s twice has the same result", kind))
	generated := *result
	err = mutate(result)
	Expect(err).NotTo(HaveOccurred())
	diff = cmp.Diff(generated, *result)
	if diff != "" {
		Fail(diff)
	}
}

//nolint:unparam
func testMutateConfigMap(m *corev1.ConfigMap, expectedFile string, mutate func(*corev1.ConfigMap) error, ignoreValues bool) {
	clearConfigMapValues := func(cm *corev1.ConfigMap) {
		newMap := map[string]string{}
		for k := range cm.Data {
			newMap[k] = "IGNORED"
		}
		cm.Data = newMap
	}

	if !ignoreValues {
		testMutate("ConfigMap", m, expectedFile, mutate)
	} else {
		By("Testing mutating the ConfigMap")
		err := mutate(m)
		Expect(err).NotTo(HaveOccurred())

		expected := &corev1.ConfigMap{}
		data, err := os.ReadFile(expectedFile)
		Expect(err).NotTo(HaveOccurred())
		err = yaml.UnmarshalStrict(data, expected)
		Expect(err).NotTo(HaveOccurred())

		c := m.DeepCopy()
		clearConfigMapValues(c)
		clearConfigMapValues(expected)

		diff := cmp.Diff(*expected, *c)
		if diff != "" {
			Fail(diff)
		}
	}
}

func testMutates[R Reconciler, O pdoknlv3.WMSWFS](reconcilerFn func() R, resource O, name string, ignoreFiles ...string) {
	inputPath := testPath(resource.Type(), name) + "input/"
	outputPath := testPath(resource.Type(), name) + "expected/"

	shouldIncludeFile := func(name string) (string, bool) {
		if slices.Contains(ignoreFiles, name) {
			return "", false
		}

		return outputPath + name, true
	}

	var fileName string
	switch resource.Type() {
	case pdoknlv3.ServiceTypeWFS:
		fileName = "wfs.yaml"
	case pdoknlv3.ServiceTypeWMS:
		fileName = "wms.yaml"
	default:
		panic("unknown servicetype")
	}

	owner := smoothoperatorv1.OwnerInfo{}

	It("Should parse the input files correctly", func() {
		data, err := readTestFile(inputPath + fileName)
		Expect(err).NotTo(HaveOccurred())
		err = yaml.UnmarshalStrict(data, &resource)
		Expect(err).NotTo(HaveOccurred())
		Expect(resource.GetName()).Should(Equal(name))

		data, err = os.ReadFile(inputPath + "ownerinfo.yaml")
		Expect(err).NotTo(HaveOccurred())
		err = yaml.UnmarshalStrict(data, &owner)
		Expect(err).NotTo(HaveOccurred())
		Expect(owner.Name).Should(Equal("owner"))

		var validationError error
		switch any(resource).(type) {
		case *pdoknlv3.WMS:
			wms := any(resource).(*pdoknlv3.WMS)
			_, validationError = wms.ValidateCreate()
		case *pdoknlv3.WFS:
			wfs := any(resource).(*pdoknlv3.WFS)
			_, validationError = wfs.ValidateCreate()
		}
		Expect(validationError).NotTo(HaveOccurred())
	})

	configMapNames := types.HashedConfigMapNames{}

	It("Should generate a correct Configmap", func() {
		cm := getBareConfigMap(resource, constants.MapserverName)
		testMutateConfigMap(cm, outputPath+"configmap-mapserver.yaml", func(cm *corev1.ConfigMap) error {
			return mutateConfigMap(reconcilerFn(), resource, cm)
		}, true)
		configMapNames.Mapserver = cm.Name
	})

	It("Should generate a correct BlobDownload Configmap", func() {
		if path, include := shouldIncludeFile("configmap-init-scripts.yaml"); include {
			cm := getBareConfigMap(resource, constants.InitScriptsName)
			testMutateConfigMap(cm, path, func(cm *corev1.ConfigMap) error {
				return mutateConfigMapBlobDownload(reconcilerFn(), resource, cm)
			}, true)
			configMapNames.InitScripts = cm.Name
		}
	})

	It("Should generate a correct MapfileGenerator Configmap", func() {
		if path, include := shouldIncludeFile("configmap-mapfile-generator.yaml"); include {
			cm := getBareConfigMap(resource, constants.MapfileGeneratorName)
			testMutateConfigMap(cm, path, func(cm *corev1.ConfigMap) error {
				return mutateConfigMapMapfileGenerator(reconcilerFn(), resource, cm, &owner)
			}, true)
			configMapNames.MapfileGenerator = cm.Name
		}
	})

	It("Should generate a correct CapabilitiesGenerator Configmap", func() {
		cm := getBareConfigMap(resource, constants.CapabilitiesGeneratorName)
		testMutateConfigMap(cm, outputPath+"configmap-capabilities-generator.yaml", func(cm *corev1.ConfigMap) error {
			return mutateConfigMapCapabilitiesGenerator(reconcilerFn(), resource, cm, &owner)
		}, true)
		configMapNames.CapabilitiesGenerator = cm.Name
	})

	if resource.Type() == pdoknlv3.ServiceTypeWMS {
		wms := any(resource).(*pdoknlv3.WMS)
		It("Should generate a correct FeatureInfo Configmap", func() {
			cm := getBareConfigMap(resource, constants.FeatureinfoGeneratorName)
			testMutateConfigMap(cm, outputPath+"configmap-featureinfo-generator.yaml", func(cm *corev1.ConfigMap) error {
				return mutateConfigMapFeatureinfoGenerator(getWMSReconciler(), wms, cm)
			}, true)
			configMapNames.FeatureInfoGenerator = cm.Name
		})

		It("Should generate a correct LegendGenerator Configmap", func() {
			cm := getBareConfigMap(resource, constants.LegendGeneratorName)
			testMutateConfigMap(cm, outputPath+"configmap-legend-generator.yaml", func(cm *corev1.ConfigMap) error {
				return mutateConfigMapLegendGenerator(getWMSReconciler(), wms, cm)
			}, true)
			configMapNames.LegendGenerator = cm.Name
		})

		It("Should generate a correct OGC webservice proxy Configmap", func() {
			cm := getBareConfigMap(resource, constants.OgcWebserviceProxyName)
			testMutateConfigMap(cm, outputPath+"configmap-ogc-webservice-proxy.yaml", func(cm *corev1.ConfigMap) error {
				return mutateConfigMapOgcWebserviceProxy(getWMSReconciler(), wms, cm)
			}, true)
			configMapNames.OgcWebserviceProxy = cm.Name
		})
	}

	It("Should generate a Deployment correctly", func() {
		testMutate("Deployment", getBareDeployment(resource), outputPath+"deployment.yaml", func(d *appsv1.Deployment) error {
			return mutateDeployment(reconcilerFn(), resource, d, configMapNames)
		})
	})

	It("Should generate a correct Service", func() {
		testMutate("Service", getBareService(resource), outputPath+"service.yaml", func(s *corev1.Service) error {
			return mutateService(reconcilerFn(), resource, s)
		})
	})

	It("Should generate a correct Headers Middleware", func() {
		testMutate("Headers Middleware", getBareCorsHeadersMiddleware(resource), outputPath+"middleware-headers.yaml", func(m *traefikiov1alpha1.Middleware) error {
			return mutateCorsHeadersMiddleware(reconcilerFn(), resource, m)
		})
	})

	It("Should generate a correct IngressRoute", func() {
		testMutate("IngressRoute", getBareIngressRoute(resource), outputPath+"ingressroute.yaml", func(i *traefikiov1alpha1.IngressRoute) error {
			return mutateIngressRoute(reconcilerFn(), resource, i)
		})
	})

	It("Should generate a correct PodDisruptionBudget", func() {
		testMutate("PodDisruptionBudget", getBarePodDisruptionBudget(resource), outputPath+"poddisruptionbudget.yaml", func(p *policyv1.PodDisruptionBudget) error {
			return mutatePodDisruptionBudget(reconcilerFn(), resource, p)
		})
	})

	It("Should generate a correct HorizontalPodAutoscaler", func() {
		testMutate("PodDisruptionBudget", getBareHorizontalPodAutoScaler(resource), outputPath+"horizontalpodautoscaler.yaml", func(h *v2.HorizontalPodAutoscaler) error {
			return mutateHorizontalPodAutoscaler(reconcilerFn(), resource, h)
		})
	})
}

func readTestFile(fileName string) ([]byte, error) {
	dat, err := os.ReadFile(fileName)
	if err != nil {
		return []byte{}, err
	}

	// Temporary check if the input file is a v2, if so, convert to v3
	dat, err = convertAndWriteIfWMSWFS(dat, fileName)
	if err != nil {
		return []byte{}, err
	}

	// Apply defaults
	un := unstructured.Unstructured{}
	err = yaml.Unmarshal(dat, &un)
	if slices.Contains([]string{"WMS", "WFS"}, un.GetKind()) {
		defaulted, err := smoothoperatorvalidation.ApplySchemaDefaults(un.Object)
		if err != nil {
			return []byte{}, err
		}

		return yaml.Marshal(defaulted)
	}

	return dat, err
}

func convertAndWriteIfWMSWFS(data []byte, fileName string) ([]byte, error) {
	un := unstructured.Unstructured{}
	err := yaml.Unmarshal(data, &un)
	if err != nil {
		return []byte{}, err
	}

	if un.GetAPIVersion() == "pdok.nl/v2beta1" {
		switch un.GetKind() {
		case "WFS":
			v2Wfs := v2beta1.WFS{}
			err = yaml.UnmarshalStrict(data, &v2Wfs)
			if err != nil {
				return []byte{}, err
			}
			v3 := pdoknlv3.WFS{}
			err = v2Wfs.ToV3(&v3)
			if err != nil {
				return []byte{}, err
			}
			data, err = yaml.Marshal(v3)
		case "WMS":
			v2Wms := v2beta1.WMS{}
			err = yaml.UnmarshalStrict(data, &v2Wms)
			if err != nil {
				return []byte{}, err
			}
			v3 := pdoknlv3.WMS{}
			v2Wms.ToV3(&v3)
			data, err = yaml.Marshal(v3)
		}

		_ = os.WriteFile(fileName, data, 0644)
	}

	return data, err
}
