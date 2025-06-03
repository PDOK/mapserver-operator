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
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

var _ = Describe("WFS Webhook", func() {
	var (
		obj       *pdoknlv3.WFS
		oldObj    *pdoknlv3.WFS
		validator WFSCustomValidator
	)

	BeforeEach(func() {
		validator = WFSCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")

		sample := &pdoknlv3.WFS{}
		err := readSample(sample)
		Expect(err).To(BeNil(), "Reading and parsing the WFS V3 sample failed")

		obj = sample.DeepCopy()
		oldObj = sample.DeepCopy()

		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
	})

	AfterEach(func() {

	})

	Context("When creating or updating WFS under Validating Webhook", func() {
		ctx := context.Background()

		It("Creates the WFS from the sample", func() {
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if there are no labels", func() {
			obj.Labels = nil
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if Url not in IngressRouteUrls", func() {
			Expect(obj.Spec.IngressRouteURLs).NotTo(BeNil())
			url, err := smoothoperatormodel.ParseURL("https://new/new")
			Expect(err).To(Not(HaveOccurred()))
			obj.Spec.Service.URL = smoothoperatormodel.URL{URL: url}
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Warns if the name contains WFS", func() {
			obj.Name += "-wfs"
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(len(warnings)).To(Equal(1))
		})

		It("Should deny creation if there is no bounding box and the defaultCRS is not EPSG:28992", func() {
			obj.Spec.Service.DefaultCrs = "EPSG:1234"
			obj.Spec.Service.Bbox = nil
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Warns if the mapfile and service/featuretype bbox are both set", func() {
			Expect(obj.Spec.Service.FeatureTypes[0].Bbox).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].Bbox.DefaultCRS).NotTo(BeNil())
			Expect(obj.Spec.Service.Bbox).NotTo(BeNil())
			obj.Spec.Service.Mapfile = &pdoknlv3.Mapfile{}
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(len(warnings)).To(Equal(2))
		})

		It("Should deny creation if SpatialID is also used as a featureType datasetMetadataID", func() {
			Expect(obj.Inspire()).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW).NotTo(BeNil())
			obj.Spec.Service.Inspire.SpatialDatasetIdentifier = obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW.MetadataIdentifier
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if serviceMetadataID is also used as a featureType datasetMetadataID", func() {
			Expect(obj.Inspire()).NotTo(BeNil())
			Expect(obj.Inspire().ServiceMetadataURL.CSW).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW).NotTo(BeNil())
			obj.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier = obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW.MetadataIdentifier
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if serviceMetadataID is the same as the SpatialID", func() {
			Expect(obj.Inspire()).NotTo(BeNil())
			Expect(obj.Inspire().ServiceMetadataURL.CSW).NotTo(BeNil())
			obj.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier = obj.Spec.Service.Inspire.SpatialDatasetIdentifier
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if service is Inspire and not all featureTypes have the same datasetMetadataID", func() {
			Expect(obj.Inspire()).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW).NotTo(BeNil())
			Expect(len(obj.Spec.Service.FeatureTypes)).To(BeNumerically(">", 1))
			Expect(obj.Spec.Service.FeatureTypes[1].DatasetMetadataURL).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[1].DatasetMetadataURL.CSW).NotTo(BeNil())
			obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW.MetadataIdentifier = ""
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if maxReplicas < minReplicas in HPAPatch", func() {
			obj.Spec.HorizontalPodAutoscalerPatch = &pdoknlv3.HorizontalPodAutoscalerPatch{
				MinReplicas: smoothoperatorutils.Pointer(int32(5)),
				MaxReplicas: smoothoperatorutils.Pointer(int32(1)),
			}
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if no ephemeralStorage is set on the mapserver container", func() {
			obj.Spec.PodSpecPatch.Containers = []corev1.Container{}
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if multiple featureTypes have the same name", func() {
			Expect(len(obj.Spec.Service.FeatureTypes)).To(BeNumerically(">", 1))
			obj.Spec.Service.FeatureTypes[1].Name = obj.Spec.Service.FeatureTypes[0].Name
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Warns if the mapfile and featuretype.tif extra settings are both set", func() {
			Expect(len(obj.Spec.Service.FeatureTypes)).To(BeNumerically(">", 1))
			Expect(obj.Spec.Service.FeatureTypes[1].Data.TIF).NotTo(BeNil())
			obj.Spec.Service.FeatureTypes[1].Data.TIF = &pdoknlv3.TIF{
				BlobKey:                     obj.Spec.Service.FeatureTypes[1].Data.TIF.BlobKey,
				Resample:                    "AVERAGE",
				Offsite:                     smoothoperatorutils.Pointer("#555555"),
				GetFeatureInfoIncludesClass: true,
			}
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(len(warnings)).To(Equal(3))
		})

		It("Should deny update if a url was changed and ingressRouteUrls = nil", func() {
			url, err := smoothoperatormodel.ParseURL("http://old/path")
			Expect(err).To(BeNil())
			obj.Spec.Service.URL = smoothoperatormodel.URL{URL: url}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(BeEmpty())

			oldObj.Spec.IngressRouteURLs = nil
			obj.Spec.IngressRouteURLs = nil
			warnings, err = validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if a ingressRouteURL was removed", func() {
			Expect(len(oldObj.Spec.IngressRouteURLs)).To(Equal(2))
			obj.Spec.IngressRouteURLs = []smoothoperatormodel.IngressRouteURL{{URL: obj.URL()}}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update url was changed but not added to ingressRouteURLs", func() {
			url, err := smoothoperatormodel.ParseURL("http://old/changed")
			Expect(err).ToNot(HaveOccurred())
			oldObj.Spec.IngressRouteURLs = nil
			obj.Spec.IngressRouteURLs = []smoothoperatormodel.IngressRouteURL{{URL: oldObj.Spec.Service.URL}}
			obj.Spec.Service.URL = smoothoperatormodel.URL{URL: url}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			obj.Spec.IngressRouteURLs = []smoothoperatormodel.IngressRouteURL{{URL: smoothoperatormodel.URL{URL: url}}}
			warnings, err = validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())

		})

		It("Should deny update if a label was removed", func() {
			for label := range obj.Labels {
				delete(obj.Labels, label)
				break
			}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if a label changed", func() {
			for label, val := range obj.Labels {
				obj.Labels[label] = val + "-newval"
				break
			}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if a label was added", func() {
			obj.Labels["new-label"] = "test"
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if an inspire block was added", func() {
			Expect(obj.Spec.Service.Inspire).NotTo(BeNil())
			oldObj.Spec.Service.Inspire = nil
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if an inspire block was removed", func() {
			Expect(oldObj.Spec.Service.Inspire).NotTo(BeNil())
			obj.Spec.Service.Inspire = nil
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})
	})

})
