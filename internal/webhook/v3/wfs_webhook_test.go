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
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("WFS Webhook", func() {
	var (
		obj       *pdoknlv3.WFS
		oldObj    *pdoknlv3.WFS
		validator WFSCustomValidator
		ownerInfo *smoothoperatorv1.OwnerInfo
	)

	BeforeEach(func() {
		validator = WFSCustomValidator{k8sClient}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")

		sample := &pdoknlv3.WFS{}
		err := readSample(sample)
		Expect(err).To(BeNil(), "Reading and parsing the WFS V3 sample failed")

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
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("metadata").Child("labels"),
				"can't be empty",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when URL not in IngressRouteURLs", func() {
			url, err := smoothoperatormodel.ParseURL("http://changed/changed")
			Expect(err).To(BeNil())
			obj.Spec.IngressRouteURLs = []smoothoperatormodel.IngressRouteURL{{URL: smoothoperatormodel.URL{URL: url}}}
			url, err = smoothoperatormodel.ParseURL("http://sample/sample")
			Expect(err).To(BeNil())
			obj.Spec.Service.URL = smoothoperatormodel.URL{URL: url}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("ingressRouteUrls"),
				fmt.Sprint(obj.Spec.IngressRouteURLs),
				fmt.Sprintf("must contain baseURL: %s", url),
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Warns if the name contains WFS", func() {
			obj.Name += "-wfs"
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(Equal(getValidationWarnings(
				obj,
				*field.NewPath("metadata").Child("name"),
				"name should not contain wfs",
				[]string{},
			)))
		})

		It("Should deny creation if there is no bounding box and the defaultCRS is not EPSG:28992", func() {
			obj.Spec.Service.DefaultCrs = "EPSG:1234"
			obj.Spec.Service.Bbox = nil
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("bbox").Child("defaultCRS"),
				"when service.defaultCRS is not 'EPSG:28992'",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Warns if the mapfile and service/featuretype bbox are both set", func() {
			Expect(obj.Spec.Service.FeatureTypes[0].Bbox).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].Bbox.DefaultCRS).NotTo(BeNil())
			Expect(obj.Spec.Service.Bbox).NotTo(BeNil())
			obj.Spec.Service.Mapfile = &pdoknlv3.Mapfile{}
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(Equal(getValidationWarnings(
				obj,
				*field.NewPath("spec").Child("service").Child("featureTypes").Index(0).Child("bbox").Child("defaultCrs"),
				"is not used when service.mapfile is configured",
				getValidationWarnings(
					obj,
					*field.NewPath("spec").Child("service").Child("bbox"),
					"is not used when service.mapfile is configured",
					[]string{},
				))))
		})

		It("Should deny Create when a otherCrs has the same crs multiple times", func() {
			crs := "EPSG:3035"
			obj.Spec.Service.OtherCrs = []string{crs, crs}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Duplicate(
				field.NewPath("spec").Child("service").Child("otherCrs").Index(1),
				crs,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should warn on creation if SpatialID is also used as a featureType datasetMetadataID", func() {
			Expect(obj.Inspire()).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW).NotTo(BeNil())
			obj.Spec.Service.Inspire.SpatialDatasetIdentifier = obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW.MetadataIdentifier
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(Equal(admission.Warnings{field.Invalid(
				field.NewPath("spec").Child("service").Child("inspire").Child("spatialDatasetIdentifier"),
				obj.Spec.Service.Inspire.SpatialDatasetIdentifier,
				"spatialDatasetIdentifier should not also be used as an datasetMetadataUrl.csw.metadataIdentifier",
			).Error()}))
		})

		It("Should deny creation if serviceMetadataID is also used as a featureType datasetMetadataID", func() {
			Expect(obj.Inspire()).NotTo(BeNil())
			Expect(obj.Inspire().ServiceMetadataURL.CSW).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL).NotTo(BeNil())
			Expect(obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW).NotTo(BeNil())
			obj.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier = obj.Spec.Service.FeatureTypes[0].DatasetMetadataURL.CSW.MetadataIdentifier
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("inspire").Child("csw").Child("metadataIdentifier"),
				obj.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier,
				"serviceMetadataUrl.csw.metadataIdentifier cannot also be used as an datasetMetadataUrl.csw.metadataIdentifier",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if serviceMetadataID is the same as the SpatialID", func() {
			Expect(obj.Inspire()).NotTo(BeNil())
			Expect(obj.Inspire().ServiceMetadataURL.CSW).NotTo(BeNil())
			obj.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier = obj.Spec.Service.Inspire.SpatialDatasetIdentifier
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("inspire").Child("csw").Child("metadataIdentifier"),
				obj.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier,
				"serviceMetadataUrl.csw.metadataIdentifier cannot also be used as the spatialDatasetIdentifier",
			))))
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
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("featureTypes[*]").Child("datasetMetadataUrl").Child("csw").Child("metadataIdentifier"),
				obj.DatasetMetadataIDs(),
				"when Inspire, all featureTypes need use the same datasetMetadataUrl.csw.metadataIdentifier",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when minReplicas are larger than maxReplicas", func() {
			obj.Spec.HorizontalPodAutoscalerPatch = &pdoknlv3.HorizontalPodAutoscalerPatch{
				MinReplicas: ptr.To(int32(10)),
				MaxReplicas: ptr.To(int32(5)),
			}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("horizontalPodAutoscaler"),
				fmt.Sprintf("minReplicas: %d, maxReplicas: %d", 10, 5),
				"maxReplicas cannot be less than minReplicas",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when mapserver container doesn't have ephemeral storage", func() {
			obj.Spec.PodSpecPatch = corev1.PodSpec{}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(field.NewPath("spec").
				Child("podSpecPatch").
				Child("containers").
				Key("mapserver").
				Child("resources").
				Child("limits").
				Child(corev1.ResourceEphemeralStorage.String()), ""))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny creation if multiple featureTypes have the same name", func() {
			Expect(len(obj.Spec.Service.FeatureTypes)).To(BeNumerically(">", 1))
			obj.Spec.Service.FeatureTypes[1].Name = obj.Spec.Service.FeatureTypes[0].Name
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Duplicate(
				field.NewPath("spec").Child("service").Child("featureTypes").Index(1).Child("name"),
				obj.Spec.Service.FeatureTypes[1].Name,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny create if the OwnerInfoRef doesn't exist", func() {
			obj.Spec.Service.OwnerInfoRef = "changed"

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.NotFound(
				field.NewPath("spec").Child("service").Child("ownerInfoRef"),
				obj.Spec.Service.OwnerInfoRef,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny create if the OwnerInfoRef misses namespaceTemplate", func() {
			ownerInfo.Spec.NamespaceTemplate = nil

			Expect(updateOwnerInfo(ctx, k8sClient, ownerInfo)).To(Succeed())
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("ownerInfoRef"),
				"spec.namespaceTemplate missing in "+ownerInfo.Name,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny create if the OwnerInfoRef misses csw metadataTemplate", func() {
			obj.Spec.Service.Inspire = &pdoknlv3.WFSInspire{Inspire: pdoknlv3.Inspire{ServiceMetadataURL: pdoknlv3.MetadataURL{CSW: &pdoknlv3.Metadata{MetadataIdentifier: "metadata"}}}}
			ownerInfo.Spec.MetadataUrls.CSW = nil

			Expect(updateOwnerInfo(ctx, k8sClient, ownerInfo)).To(Succeed())
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("ownerInfoRef"),
				"spec.metadataUrls.csw missing in "+ownerInfo.Name,
			))))
			Expect(warnings).To(BeEmpty())

			ownerInfo.Spec.MetadataUrls = nil
			Expect(updateOwnerInfo(ctx, k8sClient, ownerInfo)).To(Succeed())
			warnings, err = validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("ownerInfoRef"),
				"spec.metadataUrls.csw missing in "+ownerInfo.Name,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny create if the OwnerInfoRef misses WMS", func() {
			ownerInfo.Spec.WFS = nil
			Expect(updateOwnerInfo(ctx, k8sClient, ownerInfo)).To(Succeed())
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("ownerInfoRef"),
				"spec.WFS missing in "+ownerInfo.Name,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if a ingressRouteURL was removed", func() {
			url, err := smoothoperatormodel.ParseURL("http://new.url/path")
			Expect(err).To(BeNil())
			oldObj.Spec.IngressRouteURLs = []smoothoperatormodel.IngressRouteURL{
				{URL: obj.URL()},
				{URL: smoothoperatormodel.URL{URL: url}},
			}
			obj.Spec.IngressRouteURLs = []smoothoperatormodel.IngressRouteURL{{URL: obj.URL()}}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("ingressRouteUrls"),
				fmt.Sprint(obj.Spec.IngressRouteURLs),
				fmt.Sprintf("urls cannot be removed, missing: %s", smoothoperatormodel.IngressRouteURL{URL: smoothoperatormodel.URL{URL: url}}),
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should accept update if a url was changed when it's in ingressRouteUrls", func() {
			url, err := smoothoperatormodel.ParseURL("http://new.url/path")
			Expect(err).To(BeNil())
			oldObj.Spec.IngressRouteURLs = []smoothoperatormodel.IngressRouteURL{
				{URL: obj.URL()},
				{URL: smoothoperatormodel.URL{URL: url}},
			}
			obj.Spec.IngressRouteURLs = oldObj.Spec.IngressRouteURLs
			oldObj.Spec.Service.URL = obj.URL()
			obj.Spec.Service.URL = smoothoperatormodel.URL{URL: url}

			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if a url was changed and ingressRouteUrls = nil", func() {
			url, err := smoothoperatormodel.ParseURL("http://new.url/path")
			Expect(err).To(BeNil())
			obj.Spec.Service.URL = smoothoperatormodel.URL{URL: url}
			obj.Spec.IngressRouteURLs = nil
			oldObj.Spec.IngressRouteURLs = nil

			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Forbidden(
				field.NewPath("spec").Child("service").Child("url"),
				"is immutable",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update url was changed but not added to ingressRouteURLs", func() {
			url, err := smoothoperatormodel.ParseURL("http://new.url/path")
			Expect(err).ToNot(HaveOccurred())
			oldObj.Spec.IngressRouteURLs = nil
			obj.Spec.IngressRouteURLs = []smoothoperatormodel.IngressRouteURL{{URL: oldObj.Spec.Service.URL}}
			obj.Spec.Service.URL = smoothoperatormodel.URL{URL: url}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("ingressRouteUrls"),
				fmt.Sprint(obj.Spec.IngressRouteURLs),
				fmt.Sprintf("must contain baseURL: %s", obj.URL()),
			))))
			Expect(warnings).To(BeEmpty())

			obj.Spec.IngressRouteURLs = []smoothoperatormodel.IngressRouteURL{{URL: smoothoperatormodel.URL{URL: url}}}
			warnings, err = validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("ingressRouteUrls"),
				fmt.Sprint(obj.Spec.IngressRouteURLs),
				fmt.Sprintf("must contain baseURL: %s", oldObj.URL()),
			))))
			Expect(warnings).To(BeEmpty())

		})

		It("Should deny update if a label was removed", func() {
			oldKey := ""
			for label := range obj.Labels {
				oldKey = label
				delete(obj.Labels, label)
				break
			}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("metadata").Child("labels").Child(oldKey),
				"labels cannot be removed",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if a label changed", func() {
			oldKey := ""
			oldValue := ""
			newValue := ""
			for label, val := range obj.Labels {
				oldKey = label
				oldValue = val
				newValue = val + "-newval"
				obj.Labels[label] = newValue
				break
			}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("metadata").Child("labels").Child(oldKey),
				newValue,
				"immutable: should be: "+oldValue,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if a label was added", func() {
			newKey := "new-label"
			obj.Labels[newKey] = "test"
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Forbidden(
				field.NewPath("metadata").Child("labels").Child(newKey),
				"new labels cannot be added",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if an inspire block was added", func() {
			obj.Spec.Service.Inspire = &pdoknlv3.WFSInspire{}
			oldObj.Spec.Service.Inspire = nil
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Forbidden(
				field.NewPath("spec").Child("service").Child("inspire"),
				"cannot change from inspire to not inspire or the other way around",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if an inspire block was removed", func() {
			oldObj.Spec.Service.Inspire = &pdoknlv3.WFSInspire{}
			obj.Spec.Service.Inspire = nil
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Forbidden(
				field.NewPath("spec").Child("service").Child("inspire"),
				"cannot change from inspire to not inspire or the other way around",
			))))
			Expect(warnings).To(BeEmpty())
		})
	})

})
