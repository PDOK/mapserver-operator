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
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorvalidation "github.com/pdok/smooth-operator/pkg/validation"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Testing WFS Controller", func() {

	Context("Testing Mutate functions for Minimal WFS", func() {
		testMutates(getWFSReconciler, &pdoknlv3.WFS{}, "minimal")
	})

	Context("Testing Mutate functions for Complete WFS", func() {
		testMutates(getWFSReconciler, &pdoknlv3.WFS{}, "complete")
	})

	Context("Testing Mutate functions for WFS with prefetchData false", func() {
		testMutates(getWFSReconciler, &pdoknlv3.WFS{}, "noprefetch", "configmap-init-scripts.yaml")
	})

	Context("When reconciling a resource", func() {

		ctx := context.Background()

		inputPath := testPath(pdoknlv3.ServiceTypeWFS, "complete") + "input/"

		testWfs := pdoknlv3.WFS{}
		clusterWfs := &pdoknlv3.WFS{}

		objectKeyWfs := k8stypes.NamespacedName{}

		testOwner := smoothoperatorv1.OwnerInfo{}
		clusterOwner := &smoothoperatorv1.OwnerInfo{}

		objectKeyOwner := k8stypes.NamespacedName{}

		var expectedResources []struct {
			obj client.Object
			key k8stypes.NamespacedName
		}

		It("Should create a WFS and OwnerInfo resource on the cluster", func() {

			By("Creating a new resource for the Kind WFS")
			data, err := readTestFile(inputPath + "wfs.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = yaml.UnmarshalStrict(data, &testWfs)
			Expect(err).NotTo(HaveOccurred())
			Expect(testWfs.Name).Should(Equal("complete"))

			objectKeyWfs = k8stypes.NamespacedName{
				Namespace: testWfs.GetNamespace(),
				Name:      testWfs.GetName(),
			}

			err = k8sClient.Get(ctx, objectKeyWfs, clusterWfs)
			Expect(client.IgnoreNotFound(err)).To(Not(HaveOccurred()))
			if err != nil && apierrors.IsNotFound(err) {
				resource := testWfs.DeepCopy()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, objectKeyWfs, clusterWfs)).To(Succeed())
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
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyWfs})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create all expected resources", func() {
			expectedResources, err := getExpectedObjects(ctx, clusterWfs, true, true)
			Expect(err).NotTo(HaveOccurred())
			for _, expectedResource := range expectedResources {
				Eventually(func() bool {
					err := k8sClient.Get(ctx, k8stypes.NamespacedName{Namespace: expectedResource.GetNamespace(), Name: expectedResource.GetName()}, expectedResource)
					return Expect(err).NotTo(HaveOccurred())
				}, "10s", "1s").Should(BeTrue())
			}
		})

		It("Should successfully reconcile after a change in an owned resource", func() {
			controllerReconciler := getWFSReconciler()

			By("Getting the original Deployment")
			deployment := getBareDeployment(clusterWfs)
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

			By("Reconciling the WFS again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyWfs})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was restored")
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(*deployment.Spec.RevisionHistoryLimit).To(BeEquivalentTo(originalRevisionHistoryLimit))
			}, "10s", "1s").Should(BeTrue())
		})

		It("Should delete PodDisruptionBudget if Min and Max replicas == 1 ", func() {
			controllerReconciler := getWFSReconciler()

			By("Setting Min and Max replicas to 1")

			Expect(k8sClient.Get(ctx, objectKeyWfs, clusterWfs)).To(Succeed())

			resource := clusterWfs.DeepCopy()

			resource.Spec.HorizontalPodAutoscalerPatch.MinReplicas = ptr.To(int32(1))
			resource.Spec.HorizontalPodAutoscalerPatch.MaxReplicas = ptr.To(int32(1))

			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			podDisruptionBudget := getBarePodDisruptionBudget(resource)

			By("Reconciling the WFS")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyWfs})
			Expect(err).NotTo(HaveOccurred())

			By("Getting the PodDisruptionBudget")
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())

			Expect(k8sClient.Get(ctx, objectKeyWfs, clusterWfs)).To(Succeed())
			Expect(clusterWfs.Status.OperationResults[smoothoperatorutils.GetObjectFullName(k8sClient, podDisruptionBudget)]).To(Equal(controllerutil.OperationResult("deleted")))
		})

		It("Should not Create PodDisruptionBudget if Min and Max replicas == 1 ", func() {
			controllerReconciler := getWFSReconciler()

			By("Getting Cluster WFS Min and Max replicas to 1")
			Expect(k8sClient.Get(ctx, objectKeyWfs, clusterWfs)).To(Succeed())

			Expect(clusterWfs.HorizontalPodAutoscalerPatch().MaxReplicas).To(Equal(ptr.To(int32(1))))
			Expect(clusterWfs.HorizontalPodAutoscalerPatch().MinReplicas).To(Equal(ptr.To(int32(1))))

			podDisruptionBudget := getBarePodDisruptionBudget(clusterWfs)

			By("Reconciling the WFS")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyWfs})
			Expect(err).NotTo(HaveOccurred())

			By("Getting the PodDisruptionBudget")
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())

			Expect(k8sClient.Get(ctx, objectKeyWfs, clusterWfs)).To(Succeed())
			_, ok := clusterWfs.Status.OperationResults[smoothoperatorutils.GetObjectFullName(k8sClient, podDisruptionBudget)]
			Expect(ok).To(BeFalse())
		})

		It("Respects the TTL of the WFS", func() {
			By("Creating a new resource for the Kind WFS")

			ttlName := testWfs.GetName() + "-ttl"
			ttlWfs := testWfs.DeepCopy()
			ttlWfs.Name = ttlName
			ttlWfs.Spec.Lifecycle = &model.Lifecycle{TTLInDays: smoothoperatorutils.Pointer(int32(0))}
			objectKeyTTLWFS := client.ObjectKeyFromObject(ttlWfs)

			err := k8sClient.Get(ctx, objectKeyTTLWFS, ttlWfs)
			Expect(client.IgnoreNotFound(err)).To(Not(HaveOccurred()))
			if err != nil && apierrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, ttlWfs)).To(Succeed())
			}

			// Reconcile
			_, err = getWFSReconciler().Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyTTLWFS})
			Expect(err).To(Not(HaveOccurred()))

			// Check the WFS cannot be found anymore
			Eventually(func() bool {
				err = k8sClient.Get(ctx, objectKeyTTLWFS, ttlWfs)
				return apierrors.IsNotFound(err)
			}, "10s", "1s").Should(BeTrue())

			// Not checking owned resources because the test env does not do garbage collection
		})

		It("Should cleanup the cluster", func() {
			err := k8sClient.Get(ctx, objectKeyWfs, clusterWfs)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WFS")
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, clusterWfs))).To(Succeed())

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
			err := smoothoperatorvalidation.LoadSchemasForCRD(cfg, "default", "wfs.pdok.nl")
			Expect(err).NotTo(HaveOccurred())

			filepath := "input/wfs.yaml"
			testCases := []string{
				testPath(pdoknlv3.ServiceTypeWFS, "minimal") + filepath,
				testPath(pdoknlv3.ServiceTypeWFS, "complete") + filepath,
				testPath(pdoknlv3.ServiceTypeWFS, "noprefetch") + filepath,
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

func getWFSReconciler() *WFSReconciler {
	return &WFSReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
		Images: types.Images{
			MultitoolImage:             testImageName1,
			MapfileGeneratorImage:      testImageName2,
			MapserverImage:             testImageName3,
			CapabilitiesGeneratorImage: testImageName4,
			ApacheExporterImage:        testImageName5,
		},
	}
}
