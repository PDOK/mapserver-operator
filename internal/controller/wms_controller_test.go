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
	"os"

	"github.com/pdok/mapserver-operator/internal/controller/types"
	"github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorvalidation "github.com/pdok/smooth-operator/pkg/validation"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Testing WMS Controller", func() {

	Context("Testing Mutate functions for Minimal WMS", func() {
		testMutates(getWMSReconciler, &pdoknlv3.WMS{}, "minimal")
	})

	Context("Testing Mutate functions for Minimal WMS without prefetch", func() {
		testMutates(getWMSReconciler, &pdoknlv3.WMS{}, "noprefetch", "configmap-init-scripts.yaml")
	})

	Context("Testing Mutate functions for Minimal WMS with a custom mapfile", func() {
		testMutates(getWMSReconciler, &pdoknlv3.WMS{}, "custom-mapfile", "configmap-mapfile-generator.yaml")
	})

	Context("Testing Mutate functions for Complete WMS", func() {
		testMutates(getWMSReconciler, &pdoknlv3.WMS{}, "complete")
	})

	Context("When reconciling a resource", func() {

		ctx := context.Background()

		inputPath := testPath(pdoknlv3.ServiceTypeWMS, "minimal") + "input/"

		testWMS := pdoknlv3.WMS{}
		clusterWMS := &pdoknlv3.WMS{}

		objectKeyWMS := k8stypes.NamespacedName{}

		testOwner := smoothoperatorv1.OwnerInfo{}
		clusterOwner := &smoothoperatorv1.OwnerInfo{}

		objectKeyOwner := k8stypes.NamespacedName{}

		var expectedResources []struct {
			obj client.Object
			key k8stypes.NamespacedName
		}

		It("Should create a WMS and OwnerInfo resource on the cluster", func() {

			By("Creating a new resource for the Kind WMS")
			data, err := readTestFile(inputPath + "wms.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = yaml.UnmarshalStrict(data, &testWMS)
			Expect(err).NotTo(HaveOccurred())
			Expect(testWMS.Name).Should(Equal("minimal"))

			objectKeyWMS = k8stypes.NamespacedName{
				Namespace: testWMS.GetNamespace(),
				Name:      testWMS.GetName(),
			}

			err = k8sClient.Get(ctx, objectKeyWMS, clusterWMS)
			Expect(client.IgnoreNotFound(err)).To(Not(HaveOccurred()))
			if err != nil && apierrors.IsNotFound(err) {
				resource := testWMS.DeepCopy()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, objectKeyWMS, clusterWMS)).To(Succeed())
			}

			By("Creating a new resource for the Kind OwnerInfo")
			data, err = os.ReadFile(inputPath + "ownerinfo.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = yaml.UnmarshalStrict(data, &testOwner)
			Expect(err).NotTo(HaveOccurred())
			Expect(testOwner.Name).Should(Equal("owner"))

			objectKeyOwner = k8stypes.NamespacedName{
				Namespace: testOwner.GetNamespace(),
				Name:      testOwner.GetName(),
			}

			err = k8sClient.Get(ctx, objectKeyOwner, clusterOwner)
			Expect(client.IgnoreNotFound(err)).To(Not(HaveOccurred()))
			if err != nil && apierrors.IsNotFound(err) {
				resource := testOwner.DeepCopy()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, objectKeyOwner, clusterOwner)).To(Succeed())
			}
		})

		It("Should reconcile successfully", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyWMS})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create all expected resources", func() {
			expectedResources, err := getExpectedObjects(ctx, clusterWMS, true, true)
			Expect(err).NotTo(HaveOccurred())
			for _, expectedResource := range expectedResources {
				Eventually(func() bool {
					err := k8sClient.Get(ctx, k8stypes.NamespacedName{Namespace: expectedResource.GetNamespace(), Name: expectedResource.GetName()}, expectedResource)
					return Expect(err).NotTo(HaveOccurred())
				}, "10s", "1s").Should(BeTrue())
			}
		})

		It("Should successfully reconcile after a change in an owned resource", func() {
			controllerReconciler := getWMSReconciler()

			By("Getting the original Deployment")
			deployment := getBareDeployment(clusterWMS)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())
			originalRevisionHistoryLimit := *deployment.Spec.RevisionHistoryLimit
			expectedRevisionHistoryLimit := 99
			Expect(originalRevisionHistoryLimit).Should(Not(Equal(expectedRevisionHistoryLimit)))

			By("Altering the Deployment")
			err := k8sClient.Patch(ctx, deployment, client.RawPatch(k8stypes.MergePatchType, []byte(
				fmt.Sprintf(`{"spec": {"revisionHistoryLimit": %d}}`, expectedRevisionHistoryLimit))))
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was altered")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(*deployment.Spec.RevisionHistoryLimit).To(BeEquivalentTo(expectedRevisionHistoryLimit))
			}, "10s", "1s").Should(BeTrue())

			By("Reconciling the WMS again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyWMS})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was restored")
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(*deployment.Spec.RevisionHistoryLimit).To(BeEquivalentTo(originalRevisionHistoryLimit))
			}, "10s", "1s").Should(BeTrue())
		})

		It("Respects the TTL of the WMS", func() {
			By("Creating a new resource for the Kind WMS")

			ttlName := testWMS.GetName() + "-ttl"
			ttlWms := testWMS.DeepCopy()
			ttlWms.Name = ttlName
			ttlWms.Spec.Lifecycle = &model.Lifecycle{TTLInDays: smoothoperatorutils.Pointer(int32(0))}
			objectKeyTTLWMS := client.ObjectKeyFromObject(ttlWms)

			err := k8sClient.Get(ctx, objectKeyTTLWMS, ttlWms)
			Expect(client.IgnoreNotFound(err)).To(Not(HaveOccurred()))
			if err != nil && apierrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, ttlWms)).To(Succeed())
			}

			// Reconcile
			_, err = getWMSReconciler().Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyTTLWMS})
			Expect(err).To(Not(HaveOccurred()))

			// Check the WMS cannot be found anymore
			Eventually(func() bool {
				err = k8sClient.Get(ctx, objectKeyTTLWMS, ttlWms)
				return apierrors.IsNotFound(err)
			}, "10s", "1s").Should(BeTrue())

			// Not checking owned resources because the test env does not do garbage collection
		})

		It("Should cleanup the cluster", func() {
			err := k8sClient.Get(ctx, objectKeyWMS, clusterWMS)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WMS")
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, clusterWMS))).To(Succeed())

			err = k8sClient.Get(ctx, objectKeyOwner, clusterOwner)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance OwnerInfo")
			Expect(k8sClient.Delete(ctx, clusterOwner)).To(Succeed())

			// the testEnv does not do garbage collection (https://book.kubebuilder.io/reference/envtest#testing-considerations)
			By("Cleaning Owned Resources")
			for _, d := range expectedResources {
				err := k8sClient.Get(ctx, d.key, d.obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Delete(ctx, d.obj)).To(Succeed())
			}
		})
	})

	Context("When manually validating an incoming CRD", func() {
		It("Should not error", func() {
			err := smoothoperatorvalidation.LoadSchemasForCRD(cfg, "default", "wms.pdok.nl")
			Expect(err).NotTo(HaveOccurred())

			filepath := "input/wms.yaml"
			testCases := []string{
				testPath(pdoknlv3.ServiceTypeWMS, "minimal") + filepath,
				// testPath(pdoknlv3.ServiceTypeWMS, "complete") + filepath,
				// testPath(pdoknlv3.ServiceTypeWMS, "noprefetch") + filepath,
			}

			for _, test := range testCases {
				yamlInput, err := readTestFile(test)
				Expect(err).NotTo(HaveOccurred())

				err = smoothoperatorvalidation.ValidateSchema(string(yamlInput))
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})

func getWMSReconciler() *WMSReconciler {
	return &WMSReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
		Images: types.Images{
			MultitoolImage:             testImageName1,
			MapfileGeneratorImage:      testImageName2,
			MapserverImage:             testImageName3,
			CapabilitiesGeneratorImage: testImageName4,
			FeatureinfoGeneratorImage:  testImageName5,
			OgcWebserviceProxyImage:    testImageName6,
			ApacheExporterImage:        testImageName7,
		},
	}
}
