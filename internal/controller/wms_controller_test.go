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

// var _ = Describe("WMS Controller", func() {
//	Context("When reconciling a resource", func() {
//		const resourceName = "test-resource"
//
//		ctx := context.Background()
//
//		typeNamespacedName := types.NamespacedName{
//			Name:      resourceName,
//			Namespace: "default", // TODO(user):Modify as needed
//		}
//		wms := &pdoknlv3.WMS{}
//
//		BeforeEach(func() {
//			By("creating the custom resource for the Kind WMS")
//			err := k8sClient.Get(ctx, typeNamespacedName, wms)
//			if err != nil && errors.IsNotFound(err) {
//				resource := &pdoknlv3.WMS{
//					ObjectMeta: metav1.ObjectMeta{
//						Name:      resourceName,
//						Namespace: "default",
//					},
//					// TODO(user): Specify other spec details if needed.
//				}
//				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
//			}
//		})
//
//		AfterEach(func() {
//			// TODO(user): Cleanup logic after each test, like removing the resource instance.
//			resource := &pdoknlv3.WMS{}
//			err := k8sClient.Get(ctx, typeNamespacedName, resource)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Cleanup the specific resource instance WMS")
//			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
//		})
//		It("should successfully reconcile the resource", func() {
//			By("Reconciling the created resource")
//			controllerReconciler := &WMSReconciler{
//				Client: k8sClient,
//				Scheme: k8sClient.Scheme(),
//			}
//
//			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
//				NamespacedName: typeNamespacedName,
//			})
//			Expect(err).NotTo(HaveOccurred())
//			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
//			// Example: If you expect a certain status condition after reconciliation, verify it here.
//		})
//	})
//})
