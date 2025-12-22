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

	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

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
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(BeEmpty())
		})

		It("Should Deny Create when Labels are empty", func() {
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

		It("Warns when the name contains WMS", func() {
			obj.Name += "-wms"
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(Equal(getValidationWarnings(
				obj,
				*field.NewPath("metadata").Child("name"),
				"name should not contain wms",
				[]string{},
			)))
		})

		It("Warns when mapfile and resolution are set", func() {
			withMapfile(obj)
			obj.Spec.Service.Resolution = ptr.To(int32(5))
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(Equal(getValidationWarnings(
				obj,
				*field.NewPath("spec").Child("service").Child("resolution"),
				"not used when service.mapfile is configured",
				[]string{},
			)))
		})

		It("Warns when mapfile and defResolution are set", func() {
			withMapfile(obj)
			obj.Spec.Service.DefResolution = ptr.To(int32(5))
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(Equal(getValidationWarnings(
				obj,
				*field.NewPath("spec").Child("service").Child("defResolution"),
				"not used when service.mapfile is configured",
				[]string{},
			)))
		})

		It("Should deny Create when URL not in IngressRouteURLs", func() {
			obj.Spec.Service.Inspire = &pdoknlv3.Inspire{ServiceMetadataURL: pdoknlv3.MetadataURL{CSW: &pdoknlv3.Metadata{MetadataIdentifier: "metadata"}}}
			obj.Spec.Service.Layer.Layers[0].DatasetMetadataURL = &pdoknlv3.MetadataURL{CSW: &pdoknlv3.Metadata{MetadataIdentifier: "metadata"}}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("inspire").Child("csw").Child("metadataIdentifier"),
				"metadata",
				"serviceMetadataUrl.csw.metadataIdentifier cannot also be used as an datasetMetadataUrl.csw.metadataIdentifier",
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

		It("Should deny Create when multiple layers have the same name", func() {
			layerName := "equal"
			obj.Spec.Service.Layer.Layers[0].Name = &layerName
			obj.Spec.Service.Layer.Layers[1].Name = &layerName

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Duplicate(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(1).Child("name"),
				layerName,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when Group Layer has data set", func() {
			data := pdoknlv3.Data{}
			obj.Spec.Service.Layer.Layers[1].Data = &data

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(1).Child("data"),
				data,
				"must not be set on a GroupLayer",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Warns when mapfile and layer boundingboxes are both set", func() {
			withMapfile(obj)
			obj.Spec.Service.Layer.BoundingBoxes = []pdoknlv3.WMSBoundingBox{{
				CRS:  "",
				BBox: smoothoperatormodel.BBox{},
			}}
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(Equal(getValidationWarnings(
				obj,
				*field.NewPath("spec").Child("service").Child("layer").Child("boundingBoxes"),
				"is not used when service.mapfile is configured",
				[]string{},
			)))
		})

		It("Should deny Create when there is no layer boundingbox set for dataepsg and no custom mapfile", func() {
			obj.Spec.Service.Mapfile = nil
			obj.Spec.Service.DataEPSG = "EPSG:1234"
			obj.Spec.Service.Layer.Layers = []pdoknlv3.Layer{obj.Spec.Service.Layer.Layers[0]}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("boundingBoxes").Child("crs"),
				fmt.Sprintf("must contain a boundingBox for CRS %s when service.dataEPSG is not 'EPSG:28992'", obj.Spec.Service.DataEPSG),
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Warns when unused fields are set on a tiff connection when using a custom mapfile", func() {
			withMapfile(obj)
			obj.Spec.Service.Layer.Layers[0].Data = &pdoknlv3.Data{TIF: &pdoknlv3.TIF{
				BlobKey:                     "blobkey",
				Resample:                    "AVERAGE",
				Offsite:                     ptr.To("offsite"),
				GetFeatureInfoIncludesClass: true,
			}}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(Equal(getValidationWarnings(
				obj,
				*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("data").Child("tif").Child("getFeatureInfoIncludesClass"),
				"is not used when service.mapfile is configured",
				getValidationWarnings(
					obj,
					*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("data").Child("tif").Child("offsite"),
					"is not used when service.mapfile is configured",
					getValidationWarnings(
						obj,
						*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("data").Child("tif").Child("resample"),
						"is not used when service.mapfile is configured",
						[]string{},
					)))))
		})

		It("Should deny Create when there is a Group Layer that is not visible", func() {
			obj.Spec.Service.Layer.Layers[1].Visible = false
			obj.Spec.Service.Layer.Layers[1].Title = nil
			obj.Spec.Service.Layer.Layers[1].Abstract = nil
			obj.Spec.Service.Layer.Layers[1].Keywords = nil
			obj.Spec.Service.Layer.Layers[1].Styles[0].Title = nil

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(1).Child("visible"),
				false,
				"must be true for a "+pdoknlv3.GroupLayer,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Warns when unused fields are set on a layer that is not visible", func() {
			obj.Spec.Service.Layer.Layers[0].Visible = false
			obj.Spec.Service.Layer.Layers[0].Title = ptr.To("title")
			obj.Spec.Service.Layer.Layers[0].Abstract = ptr.To("abstract")
			obj.Spec.Service.Layer.Layers[0].Keywords = []string{"keyword"}
			obj.Spec.Service.Layer.Layers[0].DatasetMetadataURL = &pdoknlv3.MetadataURL{}
			obj.Spec.Service.Layer.Layers[0].Authority = &pdoknlv3.Authority{}
			obj.Spec.Service.Layer.Layers[0].Styles[0].Title = ptr.To("title")
			obj.Spec.Service.Layer.Layers[0].Styles[0].Abstract = ptr.To("abstract")

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(warnings).To(Equal(getValidationWarnings(
				obj,
				*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("styles").Index(0).Child("abstract"),
				"is not used when layer.visible=false", getValidationWarnings(
					obj,
					*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("styles").Index(0).Child("title"),
					"is not used when layer.visible=false",
					getValidationWarnings(
						obj,
						*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("authority"),
						"is not used when layer.visible=false",
						getValidationWarnings(
							obj,
							*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("datasetMetadataURL"),
							"is not used when layer.visible=false",
							getValidationWarnings(
								obj,
								*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("keywords"),
								"is not used when layer.visible=false",
								getValidationWarnings(
									obj,
									*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("abstract"),
									"is not used when layer.visible=false",
									getValidationWarnings(
										obj,
										*field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("title"),
										"is not used when layer.visible=false",
										[]string{},
									)))))))))
		})

		It("Should deny Create when a Layer has multiple boundingBoxes with  the same CRS", func() {
			bbox := pdoknlv3.WMSBoundingBox{
				CRS: "EPSG:28992",
				BBox: smoothoperatormodel.BBox{
					MinX: "-25000",
					MinY: "250000",
					MaxX: "280000",
					MaxY: "860000",
				},
			}
			obj.Spec.Service.Layer.BoundingBoxes = []pdoknlv3.WMSBoundingBox{bbox, bbox}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Duplicate(
				field.NewPath("spec").Child("service").Child("layer").Child("boundingBoxes").Index(1).Child("crs"),
				bbox.CRS,
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when a Layer uses the same style name multiple times", func() {
			styleName := "duplicate"
			style := pdoknlv3.Style{
				Name:          styleName,
				Title:         ptr.To("Title"),
				Visualization: obj.Spec.Service.Layer.Layers[0].Styles[0].Visualization,
			}
			obj.Spec.Service.Layer.Layers[0].Styles = []pdoknlv3.Style{style, style}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("styles").Index(1).Child("name"),
				styleName,
				"A Layer can't use the same style name multiple times",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when a Style doesn't have a title on its highest visible layer", func() {
			obj.Spec.Service.Layer.Layers[1].Styles[0].Title = nil

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(1).Child("styles").Index(0).Child("title"),
				"A Style must have a title on the highest visible Layer",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when a GroupLayer Style uses the same name as a Style from a parent Layer", func() {
			styleName := "duplicate"
			obj.Spec.Service.Layer.Styles = []pdoknlv3.Style{{Name: styleName, Title: ptr.To("title")}}
			obj.Spec.Service.Layer.Layers[1].Styles = []pdoknlv3.Style{{Name: styleName, Title: ptr.To("title")}}
			obj.Spec.Service.Layer.Layers[0].Styles[0].Name = styleName
			obj.Spec.Service.Layer.Layers[1].Layers[0].Styles[0].Name = styleName

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(1).Child("styles").Index(0).Child("name"),
				styleName,
				"A GroupLayer can't redefine the same style as a parent layer",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when a GroupLayer Style has visualization", func() {
			visualization := "file.style"
			obj.Spec.Service.Layer.Layers[1].Styles[0].Visualization = &visualization

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(1).Child("styles").Index(0).Child("visualization"),
				visualization,
				"GroupLayers must not have a visualization",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when a Style has a visualization while a custom mapfile is configured", func() {
			visualization := "file.style"
			withMapfile(obj)
			obj.Spec.Service.Layer.Layers[0].Styles[0].Visualization = &visualization

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("styles").Index(0).Child("visualization"),
				visualization,
				"is not used when spec.service.mapfile is used",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when a Data Layer has a Style with no visualization while a no custom mapfile is configured", func() {
			obj.Spec.Service.Layer.Layers[0].Styles[0].Visualization = nil

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("styles").Index(0).Child("visualization"),
				"on DataLayers when spec.service.mapfile is not used",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when a when a Visualization file is not defined in the stylingassets", func() {
			visualization := "new.style"
			obj.Spec.Service.Layer.Layers[0].Styles[0].Visualization = &visualization

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(0).Child("styles").Index(0).Child("visualization"),
				visualization,
				"must be defined be in spec.service.stylingAssets.configMapKeyRefs.Keys",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when a when a Group Layer style isn't implemented in a sub Data Layer", func() {
			obj.Spec.Service.Layer.Layers[1].Styles[0].Name = "new"

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Invalid(
				field.NewPath("spec").Child("service").Child("layer").Child("layers").Index(1).Child("layers").Index(0).Child("styles"),
				nil,
				fmt.Sprintf("dataLayer must implement style: %s, defined by a parent layer", "new"),
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny Create when there are no visible layers", func() {
			obj.Spec.Service.Layer.Layers = []pdoknlv3.Layer{obj.Spec.Service.Layer.Layers[1].Layers[0]}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("layer").Child("layers[*]").Child("visible"),
				"at least one layer must be visible",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny create if the OwnerInfoRef doesn't exist", func() {
			obj.Spec.Service.OwnerInfoRef = "changed"

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.NotFound(
				field.NewPath("spec").Child("service").Child("ownerInfoRef"),
				"changed",
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
			obj.Spec.Service.Inspire = &pdoknlv3.Inspire{ServiceMetadataURL: pdoknlv3.MetadataURL{CSW: &pdoknlv3.Metadata{MetadataIdentifier: "metadata"}}}
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
			ownerInfo.Spec.WMS = nil
			Expect(updateOwnerInfo(ctx, k8sClient, ownerInfo)).To(Succeed())
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Required(
				field.NewPath("spec").Child("service").Child("ownerInfoRef"),
				"spec.WMS missing in "+ownerInfo.Name,
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
				"is immutable, add the old and new urls to spec.ingressRouteUrls in order to change this field",
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
			obj.Spec.Service.Inspire = &pdoknlv3.Inspire{}
			oldObj.Spec.Service.Inspire = nil
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(Equal(getValidationError(obj, field.Forbidden(
				field.NewPath("spec").Child("service").Child("inspire"),
				"cannot change from inspire to not inspire or the other way around",
			))))
			Expect(warnings).To(BeEmpty())
		})

		It("Should deny update if an inspire block was removed", func() {
			oldObj.Spec.Service.Inspire = &pdoknlv3.Inspire{}
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

func withMapfile(wms *pdoknlv3.WMS) {
	wms.Spec.Service.Mapfile = &pdoknlv3.Mapfile{}
	wms.Spec.Service.Layer.Layers[0].Styles[0].Visualization = nil
	wms.Spec.Service.Layer.Layers[1].Layers[0].Styles[0].Visualization = nil
}
