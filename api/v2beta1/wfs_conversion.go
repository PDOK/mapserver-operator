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
	sharedModel "github.com/pdok/smooth-operator/model"
	"log"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
)

// ConvertTo converts this WFS (v2beta1) to the Hub version (v3).
func (src *WFS) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*pdoknlv3.WFS)
	log.Printf("ConvertTo: Converting WFS from Spoke version v2beta1 to Hub version v3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	dst.ObjectMeta = src.ObjectMeta

	// Set LifeCycle if defined
	if src.Spec.Kubernetes.Lifecycle != nil && src.Spec.Kubernetes.Lifecycle.TTLInDays != nil {
		dst.Spec.Lifecycle = &sharedModel.Lifecycle{
			TTLInDays: Pointer(int32(*src.Spec.Kubernetes.Lifecycle.TTLInDays)),
		}
	}

	if src.Spec.Kubernetes.Autoscaling != nil {
		dst.Spec.HorizontalPodAutoscalerPatch = ConverseAutoscaling(*src.Spec.Kubernetes.Autoscaling)
	}

	// TODO converse src.Spec.Kubernetes.HealthCheck when we know what the implementation in v3 will be
	if src.Spec.Kubernetes.Resources != nil {
		dst.Spec.PodSpecPatch = ConverseResources(*src.Spec.Kubernetes.Resources)
	}

	dst.Spec.Options = &pdoknlv3.Options{
		AutomaticCasing: src.Spec.Options.AutomaticCasing,
		PrefetchData:    PointerValWithDefault(src.Spec.Options.PrefetchData, false),
		IncludeIngress:  src.Spec.Options.IncludeIngress,
	}

	service := pdoknlv3.Service{
		Prefix:            "",
		BaseURL:           "https://service.pdok.nl",
		OwnerInfoRef:      "pdok",
		Title:             src.Spec.Service.Title,
		Abstract:          src.Spec.Service.Abstract,
		Keywords:          src.Spec.Service.Keywords,
		Fees:              nil,
		AccessConstraints: src.Spec.Service.AccessConstraints,
		DefaultCrs:        src.Spec.Service.DataEPSG,
		OtherCrs:          []string{},
		CountDefault:      src.Spec.Service.Maxfeatures,
		FeatureTypes:      make([]pdoknlv3.FeatureType, 0),
	}

	if src.Spec.Service.Mapfile != nil {
		service.Mapfile = &pdoknlv3.Mapfile{
			ConfigMapKeyRef: src.Spec.Service.Mapfile.ConfigMapKeyRef,
		}
	}

	if src.Spec.Service.Extent != nil && *src.Spec.Service.Extent != "" {
		service.Bbox = &pdoknlv3.Bbox{
			DefaultCRS: sharedModel.ExtentToBBox(*src.Spec.Service.Extent),
		}
	} else {
		service.Bbox = &pdoknlv3.Bbox{
			DefaultCRS: sharedModel.BBox{
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
			Language:                 "nl",
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
		DatasetMetadataURL: pdoknlv3.MetadataURL{
			CSW: &pdoknlv3.Metadata{
				MetadataIdentifier: src.DatasetMetadataIdentifier,
			},
		},
		Data: pdoknlv3.Data{},
	}

	if src.Extent != nil {
		featureTypeV3.Bbox = &pdoknlv3.FeatureBbox{
			DefaultCRS: sharedModel.ExtentToBBox(*src.Extent),
		}
	}

	if src.Data.GPKG != nil {
		featureTypeV3.Data.Gpkg = &pdoknlv3.Gpkg{
			BlobKey:      src.Data.GPKG.BlobKey,
			TableName:    src.Data.GPKG.Table,
			GeometryType: src.Data.GPKG.GeometryType,
			Columns: ConverseColumnAndAliasesV2ToColumnsWithAliasV3(
				src.Data.GPKG.Columns,
				src.Data.GPKG.Aliases,
			),
		}
	}

	if src.Data.Postgis != nil {
		featureTypeV3.Data.Postgis = &pdoknlv3.Postgis{
			TableName:    src.Data.Postgis.Table,
			GeometryType: src.Data.Postgis.GeometryType,
			Columns: ConverseColumnAndAliasesV2ToColumnsWithAliasV3(
				src.Data.Postgis.Columns,
				src.Data.Postgis.Aliases,
			),
		}
	}

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

	dst.Spec.General = General{
		Dataset:      src.ObjectMeta.Labels["dataset"],
		DatasetOwner: src.ObjectMeta.Labels["dataset-owner"],
		DataVersion:  nil,
	}

	if serviceVersion, ok := src.ObjectMeta.Labels["service-version"]; ok {
		dst.Spec.General.ServiceVersion = &serviceVersion
	}

	if theme, ok := src.ObjectMeta.Labels["theme"]; ok {
		dst.Spec.General.Theme = &theme
	}

	dst.Spec.Kubernetes = Kubernetes{}

	if src.Spec.Lifecycle != nil && src.Spec.Lifecycle.TTLInDays != nil {
		dst.Spec.Kubernetes.Lifecycle = &Lifecycle{
			TTLInDays: Pointer(int(*src.Spec.Lifecycle.TTLInDays)),
		}
	}

	// TODO - healthcheck
	if src.Spec.PodSpecPatch != nil {
		dst.Spec.Kubernetes.Resources = &src.Spec.PodSpecPatch.Containers[0].Resources
	}

	if src.Spec.HorizontalPodAutoscalerPatch != nil {
		dst.Spec.Kubernetes.Autoscaling = &Autoscaling{
			MaxReplicas: Pointer(int(src.Spec.HorizontalPodAutoscalerPatch.MaxReplicas)),
		}

		if src.Spec.HorizontalPodAutoscalerPatch.MinReplicas != nil {
			dst.Spec.Kubernetes.Autoscaling.MinReplicas = Pointer(int(*src.Spec.HorizontalPodAutoscalerPatch.MinReplicas))
		}

		if src.Spec.HorizontalPodAutoscalerPatch.Metrics != nil {
			dst.Spec.Kubernetes.Autoscaling.AverageCPUUtilization = Pointer(
				int(*src.Spec.HorizontalPodAutoscalerPatch.Metrics[0].Resource.TargetAverageUtilization),
			)
		}
	}

	if src.Spec.Options == nil {
		dst.Spec.Options = WMSWFSOptions{
			AutomaticCasing: src.Spec.Options.AutomaticCasing,
			PrefetchData:    &src.Spec.Options.PrefetchData,
			IncludeIngress:  src.Spec.Options.IncludeIngress,
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
		service.Extent = Pointer(src.Spec.Service.Bbox.DefaultCRS.ToExtent())
	} else {
		service.Extent = Pointer("-25000 250000 280000 860000")
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
	}

	for _, featureType := range src.Spec.Service.FeatureTypes {
		featureTypeV2 := FeatureType{
			Name:                      featureType.Name,
			Title:                     featureType.Title,
			Abstract:                  featureType.Abstract,
			Keywords:                  featureType.Keywords,
			DatasetMetadataIdentifier: featureType.DatasetMetadataURL.CSW.MetadataIdentifier,
			SourceMetadataIdentifier:  "",
			Data:                      Data{},
		}

		if src.Spec.Service.Inspire != nil {
			featureTypeV2.SourceMetadataIdentifier = src.Spec.Service.Inspire.SpatialDatasetIdentifier
		}

		if featureType.Bbox != nil {
			featureTypeV2.Extent = Pointer(featureType.Bbox.DefaultCRS.ToExtent())
		}

		if featureType.Data.Gpkg != nil {
			columns, aliases := ConverseColumnsWithAliasV3ToColumnsAndAliasesV2(featureType.Data.Gpkg.Columns)
			featureTypeV2.Data.GPKG = &GPKG{
				BlobKey:      featureType.Data.Gpkg.BlobKey,
				Table:        featureType.Data.Gpkg.TableName,
				GeometryType: featureType.Data.Gpkg.GeometryType,
				Columns:      columns,
				Aliases:      aliases,
			}
		}

		if featureType.Data.Postgis != nil {
			columns, aliases := ConverseColumnsWithAliasV3ToColumnsAndAliasesV2(featureType.Data.Postgis.Columns)
			featureTypeV2.Data.Postgis = &Postgis{
				Table:        featureType.Data.Postgis.TableName,
				GeometryType: featureType.Data.Postgis.GeometryType,
				Columns:      columns,
				Aliases:      aliases,
			}
		}

		service.FeatureTypes = append(service.FeatureTypes, featureTypeV2)
	}

	dst.Spec.Service = service

	return nil
}
