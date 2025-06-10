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

package v3

//nolint:revive // Complains about the dot imports
import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
)

var _ = Describe("WMS Webhook", func() {
	var (
		obj       *pdoknlv3.WMS
		oldObj    *pdoknlv3.WMS
		validator WMSCustomValidator
		ownerInfo *smoothoperatorv1.OwnerInfo
	)

	BeforeEach(func() {
		validator = WMSCustomValidator{k8sClient}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")

		sample := &pdoknlv3.WMS{}
		Expect(readSample(sample)).To(Succeed(), "Reading and parsing the WMS V3 sample failed")

		obj = sample.DeepCopy()
		oldObj = sample.DeepCopy()

		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")

		ownerInfoSample := &smoothoperatorv1.OwnerInfo{}
		Expect(readOwnerInfo(ownerInfoSample)).To(Succeed(), "Reading and parsing the Ownerinfo sample failed")
		ownerInfo = ownerInfoSample.DeepCopy()
		Expect(ownerInfo).NotTo(BeNil())
		Expect(createOwnerInfo(ctx, k8sClient, ownerInfo)).To(Succeed())

	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, ownerInfo)).To(Succeed())
	})

	Context("When creating or updating WMS under Conversion Webhook", func() {
		ctx := context.Background()

		It("Creates the WMS from the sample", func() {
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
		})

		It("Warns if the name contains WMS", func() {
			obj.Name += "-wms"
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(len(warnings)).To(BeNumerically(">", 0))
		})

		// TODO(fix/make unit tests)
		// It("Warns if the mapfile and featuretype.tif extra settings are both set", func() {
		//	Expect(len(obj.Spec.Service.FeatureTypes)).To(BeNumerically(">", 1))
		//	Expect(obj.Spec.Service.FeatureTypes[1].Data.TIF).NotTo(BeNil())
		//	obj.Spec.Service.FeatureTypes[1].Data.TIF = &pdoknlv3.TIF{
		//		BlobKey:                     obj.Spec.Service.FeatureTypes[1].Data.TIF.BlobKey,
		//		Resample:                    "AVERAGE",
		//		Offsite:                     smoothoperatorutils.Pointer("#555555"),
		//		GetFeatureInfoIncludesClass: true,
		//	}
		//	warnings, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(BeNil())
		//	Expect(len(warnings)).To(Equal(3))
		// })

		// It("Should deny creation if there are no labels", func() {
		//	obj.Labels = map[string]string{}
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		// })
		//
		// It("Should deny creation if layer names are not unique", func() {
		//	childLayers := obj.Spec.Service.Layer.Layers
		//	secondLayer := childLayers[0]
		//	obj.Spec.Service.Layer.Name = secondLayer.Name
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		// })
		//
		// It("Should deny creation if defaultCRS is not EPSG:28992 and layer has no boundingbox defined for the corresponding CRS", func() {
		//	obj.Spec.Service.DataEPSG = "EPSG:4326"
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		// })
		//
		// It("Should deny creation if layer is visible and has no value for required field title", func() {
		//	obj.Spec.Service.Layer.Title = nil
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		// })
		//
		// It("Should deny creation if layer is visible and has no value for required field abstract", func() {
		//	obj.Spec.Service.Layer.Abstract = nil
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny creation if layer is visible and has no value for required field keywords", func() {
		//	obj.Spec.Service.Layer.Keywords = nil
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny creation if there is a visible layer without a style title", func() {
		//	nestedLayers1 := obj.Spec.Service.Layer.Layers
		//	nestedLayers2 := nestedLayers1[0].Layers
		//	nestedLayers2[0].Styles[0].Title = nil
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny creation if layer has parent layer with same style name as child layer", func() {
		//	nestedLayers1 := obj.Spec.Service.Layer.Layers
		//	nestedLayers2 := nestedLayers1[0].Layers
		//	nestedLayers1[0].Styles = []pdoknlv3.Style{{Name: nestedLayers2[0].Styles[0].Name}}
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny creation if datalayer has style without visualization but there is no mapfile set", func() {
		//	nestedLayers1 := obj.Spec.Service.Layer.Layers
		//	nestedLayers2 := nestedLayers1[0].Layers
		//	nestedLayers2[0].Styles[0].Visualization = nil
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny creation if datalayer has style with visualization but there is also a mapfile set", func() {
		//	obj.Spec.Service.Mapfile = &pdoknlv3.Mapfile{ConfigMapKeyRef: corev1.ConfigMapKeySelector{Key: "mapfile.map"}}
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny creation if grouplayer is not visible", func() {
		//	nestedLayers1 := obj.Spec.Service.Layer.Layers
		//	nestedLayers1[0].Visible = false
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny creation if grouplayer has data", func() {
		//	nestedLayers1 := obj.Spec.Service.Layer.Layers
		//	nestedLayers1[0].Data = &pdoknlv3.Data{}
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny creation if grouplayer has style with visualization", func() {
		//	nestedLayers1 := obj.Spec.Service.Layer.Layers
		//	nestedLayers1[0].Styles = []pdoknlv3.Style{{Visualization: smoothoperatorutils.Pointer("visualization.style")}}
		//	_, err := validator.ValidateCreate(ctx, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny update if a label changed", func() {
		//	for label, val := range obj.Labels {
		//		obj.Labels[label] = val + "-newval"
		//		break
		//	}
		//	_, err := validator.ValidateUpdate(ctx, oldObj, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny update if a label was removed", func() {
		//	for label := range obj.Labels {
		//		delete(obj.Labels, label)
		//		break
		//	}
		//	_, err := validator.ValidateUpdate(ctx, oldObj, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny update if a label was added", func() {
		//	obj.Labels["new-label"] = "test"
		//	_, err := validator.ValidateUpdate(ctx, oldObj, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny update if an inspire block was added", func() {
		//	oldObj.Spec.Service.Inspire = nil
		//	_, err := validator.ValidateUpdate(ctx, oldObj, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny update if an inspire block was removed", func() {
		//	obj.Spec.Service.Inspire = nil
		//	_, err := validator.ValidateUpdate(ctx, oldObj, obj)
		//	Expect(err).To(HaveOccurred())
		//})
		//
		//It("Should deny creation if there are no visible layers", func() {
		//	obj.Spec.Service.Layer.Layers = nil
		//	obj.Spec.Service.Layer.Visible = false
		//
		//	_, err := validator.ValidateUpdate(ctx, oldObj, obj)
		//	Expect(err).To(HaveOccurred())
		//})
	})

})
