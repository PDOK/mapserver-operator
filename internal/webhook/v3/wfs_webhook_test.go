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
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
		})

		It("Warns if the name contains WFS", func() {
			obj.Name += "-wfs"
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(BeNil())
			Expect(len(warnings)).To(BeNumerically(">", 0))
		})

		It("Should deny creation if there are no labels", func() {
			obj.Labels = map[string]string{}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
		})

		It("Should deny creation if there is no bounding box and the defaultCRS is not EPSG:28992", func() {
			obj.Spec.Service.DefaultCrs = "EPSG:1234"
			obj.Spec.Service.Bbox = nil
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
		})

		It("Should deny update if a label changed", func() {
			for label, val := range obj.Labels {
				obj.Labels[label] = val + "-newval"
				break
			}
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})

		It("Should deny update if a label was removed", func() {
			for label := range obj.Labels {
				delete(obj.Labels, label)
				break
			}
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})

		It("Should deny update if a label was added", func() {
			obj.Labels["new-label"] = "test"
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})

		It("Should deny update if an inspire block was added", func() {
			oldObj.Spec.Service.Inspire = nil
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})

		It("Should deny update if an inspire block was removed", func() {
			obj.Spec.Service.Inspire = nil
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})
	})

})
