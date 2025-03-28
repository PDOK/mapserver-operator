/*
MIT License

Copyright (c) 2024 Publieke Dienstverlening op de Kaart

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"context"
	"fmt"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	wfsResourceName       = "wfs-resource"
	ownerInfoResourceName = "pdok"
	namespace             = "default"
	testImageName1        = "test.test/image:test1"
	testImageName2        = "test.test/image:test2"
	testImageName3        = "test.test/image:test1"
	testImageName4        = "test.test/image:test2"
)

var _ = Describe("WFS Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		// Setup variables for unique Atom resource per It node
		counter := 1
		sampleWfs := readV3Sample()
		var typeNamespacedNameWfs types.NamespacedName

		typeNamespacedNameOwnerInfo := types.NamespacedName{
			Namespace: namespace,
			Name:      ownerInfoResourceName,
		}
		ownerInfo := &smoothoperatorv1.OwnerInfo{}

		BeforeEach(func() {
			// Create a unique Atom resource for every It node to prevent unexpected resource state caused by finalizers
			sampleWfs = getUniqueFullAtom(counter)
			typeNamespacedNameWfs = getUniqueWfsTypeNamespacedName(counter)
			counter++

			By("creating the custom resource for the Kind WFS")
			err := k8sClient.Get(ctx, typeNamespacedNameWfs, sampleWfs)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, sampleWfs)).To(Succeed())
			}

			By("creating the custom resource for the Kind OwnerInfo")
			err = k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfo)
			if err != nil && errors.IsNotFound(err) {
				resource := &smoothoperatorv1.OwnerInfo{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      ownerInfoResourceName,
					},
					Spec: smoothoperatorv1.OwnerInfoSpec{
						MetadataUrls: smoothoperatorv1.MetadataUrls{
							CSW: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
							},
							OpenSearch: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/opensearch/dut/{{identifier}}/OpenSearchDescription.xml",
							},
							HTML: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/catalog.search#/metadata/{{identifier}}",
							},
						},
						WFS: smoothoperatorv1.WFS{
							ServiceProvider: smoothoperatorv1.ServiceProvider{
								ProviderName: "pdok",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfo)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := sampleWfs
			err := k8sClient.Get(ctx, typeNamespacedNameWfs, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WFS")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := getReconciler()

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedNameWfs,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})

		//It("Should create correct configMap manifest.", func() {
		//	controllerReconciler := getReconciler()
		//
		//	By("Reconciling the WFS and checking the configMap")
		//	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
		//	Expect(err).NotTo(HaveOccurred())
		//	err = k8sClient.Get(ctx, typeNamespacedNameWfs, sampleWfs)
		//	Expect(err).NotTo(HaveOccurred())
		//	Expect(sampleWfs.Finalizers).To(ContainElement(finalizerName))
		//	_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
		//	Expect(err).NotTo(HaveOccurred())
		//
		//	configMap := GetBareConfigMap(sampleWfs)
		//	Eventually(func() bool {
		//		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(configMap), configMap)
		//		return Expect(err).NotTo(HaveOccurred())
		//	}, "10s", "1s").Should(BeTrue())
		//
		//	// TODO add tests for specific fields
		//})

		It("Should create correct middlewareCorsHeaders manifest.", func() {
			controllerReconciler := getReconciler()

			By("Reconciling the WFS and checking the middlewareCorsHeaders")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameWfs, sampleWfs)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleWfs.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
			Expect(err).NotTo(HaveOccurred())

			middlewareCorsHeaders := GetBareCorsHeadersMiddleware(sampleWfs)
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(middlewareCorsHeaders), middlewareCorsHeaders)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			// TODO add tests for specific fields
			//Expect(middlewareCorsHeaders.Name).Should(Equal("test-atom-6-atom-cors-headers"))
			//Expect(middlewareCorsHeaders.Namespace).Should(Equal("default"))
			//Expect(middlewareCorsHeaders.Labels["app"]).Should(Equal("atom-service"))
			//Expect(middlewareCorsHeaders.Labels["dataset"]).Should(Equal("test-dataset"))
			//Expect(middlewareCorsHeaders.Labels["dataset-owner"]).Should(Equal("test-datasetowner"))
			//Expect(middlewareCorsHeaders.Labels["service-type"]).Should(Equal("atom"))
			//Expect(middlewareCorsHeaders.Spec.Headers.FrameDeny).Should(Equal(true))
			//Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Headers"]).Should(Equal("Content-Type"))
			//Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Method"]).Should(Equal("GET, HEAD, OPTIONS"))
			//Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Origin"]).Should(Equal("*"))
		})

		It("Should create correct podDisruptionBudget manifest.", func() {
			controllerReconciler := getReconciler()

			By("Reconciling the WFS and checking the podDisruptionBudget")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameWfs, sampleWfs)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleWfs.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
			Expect(err).NotTo(HaveOccurred())

			podDisruptionBudget := GetBarePodDisruptionBudget(sampleWfs)
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			// TODO add tests for specific fields
		})

		It("Should create correct horizontalPodAutoScaler manifest.", func() {
			controllerReconciler := getReconciler()

			By("Reconciling the WFS and checking the horizontalPodAutoScaler")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameWfs, sampleWfs)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleWfs.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
			Expect(err).NotTo(HaveOccurred())

			autoscaler := GetBareHorizontalPodAutoScaler(sampleWfs)
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(autoscaler), autoscaler)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			// TODO add tests for specific fields
		})
	})
})

func getUniqueFullAtom(counter int) *pdoknlv3.WFS {
	sample := readV3Sample()

	sample.Name = getUniqueWfsResourceName(counter)
	sample.Namespace = namespace
	sample.Spec.Service.OwnerInfoRef = ownerInfoResourceName

	return sample
}

func getUniqueWfsTypeNamespacedName(counter int) types.NamespacedName {
	return types.NamespacedName{
		Name:      getUniqueWfsResourceName(counter),
		Namespace: namespace,
	}
}

func getUniqueWfsResourceName(counter int) string {
	return fmt.Sprintf("%s-%v", wfsResourceName, counter)
}

func getReconciler() *WFSReconciler {
	return &WFSReconciler{
		Client:                     k8sClient,
		Scheme:                     k8sClient.Scheme(),
		MultitoolImage:             testImageName1,
		MapfileGeneratorImage:      testImageName2,
		MapserverImage:             testImageName3,
		CapabilitiesGeneratorImage: testImageName4,
	}
}

func readV3Sample() *pdoknlv3.WFS {
	yamlFile, err := os.ReadFile("../../config/samples/v3_wfs.yaml")
	if err != nil {
		panic(err)
	}

	wfs := &pdoknlv3.WFS{}
	err = yaml.Unmarshal(yamlFile, wfs)
	if err != nil {
		panic(err)
	}

	return wfs
}
