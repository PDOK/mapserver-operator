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

package v2beta1

import (
	"log"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"

	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this WFS (v2beta1) to the Hub version (v3).
func (src *WFS) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*pdoknlv3.WFS)
	log.Printf("ConvertTo: Converting WFS from Spoke version v2beta1 to Hub version v3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	return src.ToV3(dst)
}

//nolint:gosec
func (src *WFS) ToV3(dst *pdoknlv3.WFS) error {
	dst.ObjectMeta = src.ObjectMeta

	// Set LifeCycle if defined
	if src.Spec.Kubernetes.Lifecycle != nil && src.Spec.Kubernetes.Lifecycle.TTLInDays != nil {
		dst.Spec.Lifecycle = &smoothoperatormodel.Lifecycle{
			TTLInDays: smoothoperatorutils.Pointer(int32(*src.Spec.Kubernetes.Lifecycle.TTLInDays)),
		}
	}

	if src.Spec.Kubernetes.Autoscaling != nil {
		dst.Spec.HorizontalPodAutoscalerPatch = ConvertAutoscaling(*src.Spec.Kubernetes.Autoscaling)
	}

	if src.Spec.Kubernetes.Resources != nil {
		dst.Spec.PodSpecPatch = ConvertResources(*src.Spec.Kubernetes.Resources)
	}

	dst.Spec.Options = ConvertOptionsV2ToV3(src.Spec.Options)

	if src.Spec.Kubernetes.HealthCheck != nil {
		dst.Spec.HealthCheck = &pdoknlv3.HealthCheckWFS{
			Querystring: *src.Spec.Kubernetes.HealthCheck.Querystring,
			Mimetype:    *src.Spec.Kubernetes.HealthCheck.Mimetype,
		}
	}

	url, err := CreateBaseURL(pdoknlv3.GetHost(true), "wfs", src.Spec.General)
	if err != nil {
		return err
	}

	service := pdoknlv3.WFSService{
		Prefix:            src.Spec.General.Dataset,
		URL:               *url,
		OwnerInfoRef:      "pdok",
		Title:             src.Spec.Service.Title,
		Abstract:          src.Spec.Service.Abstract,
		Keywords:          src.Spec.Service.Keywords,
		Fees:              nil,
		AccessConstraints: src.Spec.Service.AccessConstraints,
		DefaultCrs:        src.Spec.Service.DataEPSG,
		OtherCrs: []string{
			"EPSG::25831",
			"EPSG::25832",
			"EPSG::3034",
			"EPSG::3035",
			"EPSG::3857",
			"EPSG::4258",
			"EPSG::4326",
		},
		CountDefault: src.Spec.Service.Maxfeatures,
		FeatureTypes: make([]pdoknlv3.FeatureType, 0),
	}

	if src.Spec.Service.Mapfile != nil {
		service.Mapfile = &pdoknlv3.Mapfile{
			ConfigMapKeyRef: src.Spec.Service.Mapfile.ConfigMapKeyRef,
		}
	}

	if src.Spec.Service.Extent != nil && *src.Spec.Service.Extent != "" {
		service.Bbox = &pdoknlv3.Bbox{
			DefaultCRS: smoothoperatormodel.ExtentToBBox(*src.Spec.Service.Extent),
		}
	} else {
		service.Bbox = &pdoknlv3.Bbox{
			DefaultCRS: smoothoperatormodel.BBox{
				MinX: "-25000",
				MaxX: "280000",
				MinY: "250000",
				MaxY: "860000",
			},
		}
	}

	// TODO - where to place the MetadataIdentifier and FeatureTypes[0].SourceMetadataIdentifier if the service is not inspire?
	if src.Spec.Service.Inspire {
		service.Inspire = &pdoknlv3.Inspire{
			ServiceMetadataURL: pdoknlv3.MetadataURL{
				CSW: &pdoknlv3.Metadata{
					MetadataIdentifier: src.Spec.Service.MetadataIdentifier,
				},
			},
			SpatialDatasetIdentifier: src.Spec.Service.FeatureTypes[0].SourceMetadataIdentifier,
			Language:                 "dut",
		}
	}

	for _, featureType := range src.Spec.Service.FeatureTypes {
		service.FeatureTypes = append(service.FeatureTypes, convertV2FeatureTypeToV3(featureType))
	}

	dst.Spec.Service = service

	return nil
}

func convertV2FeatureTypeToV3(src FeatureType) pdoknlv3.FeatureType {
	featureTypeV3 := pdoknlv3.FeatureType{
		Name:     src.Name,
		Title:    src.Title,
		Abstract: src.Abstract,
		Keywords: src.Keywords,
		DatasetMetadataURL: &pdoknlv3.MetadataURL{
			CSW: &pdoknlv3.Metadata{
				MetadataIdentifier: src.DatasetMetadataIdentifier,
			},
		},
		Data: pdoknlv3.Data{},
	}

	if src.Extent != nil {
		featureTypeV3.Bbox = &pdoknlv3.FeatureBbox{
			DefaultCRS: smoothoperatormodel.ExtentToBBox(*src.Extent),
		}
	}

	featureTypeV3.Data = ConvertV2DataToV3(src.Data)

	return featureTypeV3
}

// ConvertFrom converts the Hub version (v3) to this WFS (v2beta1).
//
//nolint:revive
func (dst *WFS) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*pdoknlv3.WFS)
	log.Printf("ConvertFrom: Converting WFS from Hub version v3 to Spoke version v2beta1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.General = LabelsToV2General(src.ObjectMeta.Labels)

	dst.Spec.Kubernetes = NewV2KubernetesObject(src.Spec.Lifecycle, src.Spec.PodSpecPatch, src.Spec.HorizontalPodAutoscalerPatch)

	dst.Spec.Options = ConvertOptionsV3ToV2(src.Spec.Options)

	if src.Spec.HealthCheck != nil {
		dst.Spec.Kubernetes.HealthCheck = &HealthCheck{
			Querystring: &src.Spec.HealthCheck.Querystring,
			Mimetype:    &src.Spec.HealthCheck.Mimetype,
		}
	}

	service := WFSService{
		Title:             src.Spec.Service.Title,
		Abstract:          src.Spec.Service.Abstract,
		Keywords:          src.Spec.Service.Keywords,
		AccessConstraints: src.Spec.Service.AccessConstraints,
		DataEPSG:          src.Spec.Service.DefaultCrs,
		Maxfeatures:       src.Spec.Service.CountDefault,
		Authority: Authority{
			Name: "",
			URL:  "",
		},
	}

	if src.Spec.Service.Bbox != nil {
		service.Extent = smoothoperatorutils.Pointer(src.Spec.Service.Bbox.DefaultCRS.ToExtent())
	} else {
		service.Extent = smoothoperatorutils.Pointer("-25000 250000 280000 860000")
	}

	if src.Spec.Service.Mapfile != nil {
		service.Mapfile = &Mapfile{
			ConfigMapKeyRef: src.Spec.Service.Mapfile.ConfigMapKeyRef,
		}
	}

	if src.Spec.Service.Inspire != nil {
		service.Inspire = true
		service.MetadataIdentifier = src.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier
	} else {
		service.Inspire = false
		// TODO unable to fill in MetadataIdentifier here until we know how to handle non inspire services
	}

	for _, featureType := range src.Spec.Service.FeatureTypes {
		featureTypeV2 := FeatureType{
			Name:                      featureType.Name,
			Title:                     featureType.Title,
			Abstract:                  featureType.Abstract,
			Keywords:                  featureType.Keywords,
			DatasetMetadataIdentifier: featureType.DatasetMetadataURL.CSW.MetadataIdentifier,
			SourceMetadataIdentifier:  "",
			Data:                      ConvertV3DataToV2(featureType.Data),
		}

		if src.Spec.Service.Inspire != nil {
			featureTypeV2.SourceMetadataIdentifier = src.Spec.Service.Inspire.SpatialDatasetIdentifier
		}

		if featureType.Bbox != nil {
			featureTypeV2.Extent = smoothoperatorutils.Pointer(featureType.Bbox.DefaultCRS.ToExtent())
		}

		service.FeatureTypes = append(service.FeatureTypes, featureTypeV2)
	}

	dst.Spec.Service = service

	return nil
}
