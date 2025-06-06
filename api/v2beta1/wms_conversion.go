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
	"errors"
	"log"
	"strconv"
	"strings"

	"k8s.io/utils/ptr"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"

	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

const ServiceMetatdataIdentifierAnnotation = "pdok.nl/wms-service-metadata-uuid"

// ConvertTo converts this WMS (v2beta1) to the Hub version (v3).
func (src *WMS) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*pdoknlv3.WMS)
	log.Printf("ConvertTo: Converting WMS from Spoke version v2beta1 to Hub version v3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)
	return src.ToV3(dst)
}

//nolint:gosec,cyclop,funlen
func (src *WMS) ToV3(target *pdoknlv3.WMS) error {
	dst := target

	dst.ObjectMeta = src.ObjectMeta
	if dst.Annotations == nil {
		dst.Annotations = make(map[string]string)
	}

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
	dst.Spec.HealthCheck = convertHealthCheckToV3(src.Spec.Kubernetes.HealthCheck)

	url, err := CreateBaseURL(pdoknlv3.GetHost(true), "wms", src.Spec.General)
	if err != nil {
		return err
	}

	accessConstraints, err := url.Parse("https://creativecommons.org/publicdomain/zero/1.0/deed.nl")
	if err != nil {
		return err
	}
	if src.Spec.Service.AccessConstraints != nil {
		accessConstraints, err = url.Parse(*src.Spec.Service.AccessConstraints)
		if err != nil {
			return err
		}
	}

	service := pdoknlv3.WMSService{BaseService: pdoknlv3.BaseService{
		Prefix:            src.Spec.General.Dataset,
		URL:               *url,
		OwnerInfoRef:      "pdok",
		Title:             src.Spec.Service.Title,
		Abstract:          src.Spec.Service.Abstract,
		Keywords:          src.Spec.Service.Keywords,
		AccessConstraints: smoothoperatormodel.URL{URL: accessConstraints},
	},
		Inspire:       nil,
		MaxSize:       nil,
		Resolution:    nil,
		DefResolution: nil,
		DataEPSG:      src.Spec.Service.DataEPSG,
		Layer:         src.Spec.Service.MapLayersToV3(),
	}

	if src.Spec.Service.Maxsize != nil {
		service.MaxSize = smoothoperatorutils.Pointer(int32(*src.Spec.Service.Maxsize))
	}

	if src.Spec.Service.Resolution != nil {
		service.Resolution = smoothoperatorutils.Pointer(int32(*src.Spec.Service.Resolution))
	}

	if src.Spec.Service.DefResolution != nil {
		service.DefResolution = smoothoperatorutils.Pointer(int32(*src.Spec.Service.DefResolution))
	}

	if src.Spec.Service.Mapfile != nil {
		service.Mapfile = &pdoknlv3.Mapfile{
			ConfigMapKeyRef: src.Spec.Service.Mapfile.ConfigMapKeyRef,
		}
	}

	if src.Spec.Service.Inspire {
		service.Inspire = &pdoknlv3.Inspire{
			ServiceMetadataURL: pdoknlv3.MetadataURL{
				CSW: &pdoknlv3.Metadata{
					MetadataIdentifier: src.Spec.Service.MetadataIdentifier,
				},
			},
			Language: "dut",
		}
	} else {
		// Annotation to be able to convert back to v2
		dst.Annotations[ServiceMetatdataIdentifierAnnotation] = src.Spec.Service.MetadataIdentifier
	}

	if src.Spec.Service.StylingAssets != nil {
		service.StylingAssets = &pdoknlv3.StylingAssets{
			BlobKeys:      src.Spec.Service.StylingAssets.BlobKeys,
			ConfigMapRefs: []pdoknlv3.ConfigMapRef{},
		}

		for _, cm := range src.Spec.Service.StylingAssets.ConfigMapRefs {
			service.StylingAssets.ConfigMapRefs = append(service.StylingAssets.ConfigMapRefs, pdoknlv3.ConfigMapRef{
				Name: cm.Name,
				Keys: cm.Keys,
			})
		}
	}

	dst.Spec.Service = service
	return nil
}

func convertHealthCheckToV3(v2 *HealthCheck) *pdoknlv3.HealthCheckWMS {
	if v2 != nil {
		switch {
		case v2.Querystring != nil:
			return &pdoknlv3.HealthCheckWMS{
				Querystring: v2.Querystring,
				Mimetype:    v2.Mimetype,
			}
		case v2.Boundingbox != nil:
			return &pdoknlv3.HealthCheckWMS{
				Boundingbox: smoothoperatorutils.Pointer(smoothoperatormodel.ExtentToBBox(strings.ReplaceAll(*v2.Boundingbox, ",", " "))),
			}
		}
	}

	return nil
}

// ConvertFrom converts the Hub version (v3) to this WMS (v2beta1).
//
//nolint:revive
func (dst *WMS) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*pdoknlv3.WMS)
	log.Printf("ConvertFrom: Converting WMS from Hub version v3 to Spoke version v2beta1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.General = LabelsToV2General(src.ObjectMeta.Labels)

	dst.Spec.Kubernetes = NewV2KubernetesObject(src.Spec.Lifecycle, src.Spec.PodSpecPatch, src.Spec.HorizontalPodAutoscalerPatch)
	dst.Spec.Kubernetes.HealthCheck = convertHealthCheckToV2(src.Spec.HealthCheck)

	dst.Spec.Options = ConvertOptionsV3ToV2(src.Spec.Options)

	service := WMSService{
		Title:              src.Spec.Service.Title,
		Abstract:           src.Spec.Service.Abstract,
		Keywords:           src.Spec.Service.Keywords,
		AccessConstraints:  ptr.To(src.Spec.Service.AccessConstraints.String()),
		Extent:             nil,
		DataEPSG:           src.Spec.Service.DataEPSG,
		Layers:             []WMSLayer{},
		MetadataIdentifier: "00000000-0000-0000-0000-000000000000",
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

	uuid, ok := src.Annotations[ServiceMetatdataIdentifierAnnotation]
	if service.MetadataIdentifier == "00000000-0000-0000-0000-000000000000" && ok {
		service.MetadataIdentifier = uuid
	}

	if src.Spec.Service.DefResolution != nil {
		service.DefResolution = smoothoperatorutils.Pointer(int(*src.Spec.Service.DefResolution))
	}

	if src.Spec.Service.Resolution != nil {
		service.Resolution = smoothoperatorutils.Pointer(int(*src.Spec.Service.Resolution))
	}

	if src.Spec.Service.StylingAssets != nil {
		service.StylingAssets = &StylingAssets{
			BlobKeys:      src.Spec.Service.StylingAssets.BlobKeys,
			ConfigMapRefs: []ConfigMapRef{},
		}

		for _, cm := range src.Spec.Service.StylingAssets.ConfigMapRefs {
			service.StylingAssets.ConfigMapRefs = append(service.StylingAssets.ConfigMapRefs, ConfigMapRef{
				Name: cm.Name,
				Keys: cm.Keys,
			})
		}
	}

	if v3Authority := src.GetAuthority(); v3Authority != nil {
		service.Authority = Authority{
			Name: v3Authority.Name,
			URL:  v3Authority.URL,
		}
	}

	if src.Spec.Service.MaxSize != nil {
		service.Maxsize = smoothoperatorutils.Pointer(float64(*src.Spec.Service.MaxSize))
	}

	service.Layers = mapV3LayerToV2Layers(src.Spec.Service.Layer, nil, src.Spec.Service.DataEPSG)

	// Create BBox that combines all layer bounding boxes
	for _, l := range service.Layers {
		if l.Extent != nil {
			if service.Extent == nil {
				service.Extent = l.Extent
			} else {
				bbox := smoothoperatorutils.Pointer(smoothoperatormodel.ExtentToBBox(*service.Extent)).DeepCopy()
				bbox.Combine(smoothoperatormodel.ExtentToBBox(*l.Extent))
				service.Extent = smoothoperatorutils.Pointer(bbox.ToExtent())
			}
		}
	}

	dst.Spec.Service = service

	return nil
}

func convertHealthCheckToV2(v3 *pdoknlv3.HealthCheckWMS) *HealthCheck {
	if v3 != nil {
		switch {
		case v3.Querystring != nil:
			return &HealthCheck{
				Querystring: v3.Querystring,
				Mimetype:    v3.Mimetype,
			}
		case v3.Boundingbox != nil:
			return &HealthCheck{
				Boundingbox: smoothoperatorutils.Pointer(strings.ReplaceAll(v3.Boundingbox.ToExtent(), " ", ",")),
			}
		}
	}

	return nil
}

func (v2Service WMSService) GetTopLayer() (*WMSLayer, error) {
	// Only one layer defined that has data
	if len(v2Service.Layers) == 1 && v2Service.Layers[0].Data != nil {
		return nil, nil
	}

	// If all layers are groupless there is no toplayer
	allGroupless := true
	for _, layer := range v2Service.Layers {
		if layer.Group != nil && *layer.Group != "" {
			allGroupless = false
			break
		}
	}
	if allGroupless {
		return nil, nil
	}

	// Some layers have groups defined.
	// That means that there must be one layer without a group, that's the top layer
	for _, layer := range v2Service.Layers {
		if layer.Group == nil || *layer.Group == "" {
			return &layer, nil
		}
	}

	return nil, errors.New("unable to detect the toplayer of this WMS service")
}

// MapLayersToV3
func (v2Service WMSService) MapLayersToV3() pdoknlv3.Layer {
	// Creates map of Groups: layers in that group
	// and a list of all layers without a group
	groupedLayers := map[string][]pdoknlv3.Layer{}
	var notGroupedLayers []pdoknlv3.Layer
	for _, layer := range v2Service.Layers {
		if layer.Group == nil {
			notGroupedLayers = append(notGroupedLayers, layer.MapToV3(v2Service))
		} else {
			groupedLayers[*layer.Group] = append(groupedLayers[*layer.Group], layer.MapToV3(v2Service))
		}
	}

	// if a topLayer is defined in the v2 it be the only layer without a group
	// and there are other layers that have the topLayer as their group
	// and at least one of those layers is itself a group layer
	var topLayer *pdoknlv3.Layer
	if _, ok := groupedLayers[*notGroupedLayers[0].Name]; ok && len(notGroupedLayers) == 1 {
		subLayers := groupedLayers[*notGroupedLayers[0].Name]
		ok := false
		for _, layer := range subLayers {
			if _, ok = groupedLayers[*layer.Name]; ok {
				break
			}
		}

		if ok {
			topLayer = &notGroupedLayers[0]
		}
	}

	var middleLayers []pdoknlv3.Layer

	// if the topLayer is not defined in the v2 layers
	// it needs to be created with defaults from the service
	// and in this case the middleLayers are all layers without a group
	if topLayer == nil {
		boundingBoxes := make([]pdoknlv3.WMSBoundingBox, 0)
		if v2Service.Extent != nil {
			bboxStringList := strings.Split(*v2Service.Extent, " ")
			bbox := pdoknlv3.WMSBoundingBox{
				CRS: v2Service.DataEPSG,
				BBox: smoothoperatormodel.BBox{
					MinX: bboxStringList[0],
					MaxX: bboxStringList[2],
					MinY: bboxStringList[1],
					MaxY: bboxStringList[3],
				},
			}
			boundingBoxes = append(boundingBoxes, bbox)
		}

		topLayer = &pdoknlv3.Layer{
			Title:         &v2Service.Title,
			Abstract:      &v2Service.Abstract,
			Keywords:      v2Service.Keywords,
			Layers:        []pdoknlv3.Layer{},
			BoundingBoxes: boundingBoxes,
			Visible:       true,
		}

		// adding the bottom layers to the middle layers they are grouped by
		for _, layer := range notGroupedLayers {
			bottomLayers := groupedLayers[*layer.Name]
			layer.Layers = bottomLayers
			middleLayers = append(middleLayers, layer)
		}
	}

	// if the topLayer is defined in the v2 layers
	// meaning the topLayer has a name at this point
	// then the middleLayers are all layers that had the topLayer name as their group
	// and the bottomLayers are all layers that had a middleLayer as a group
	if topLayer.Name != nil {
		for _, layer := range groupedLayers[*topLayer.Name] {
			bottomLayers := groupedLayers[*layer.Name]
			layer.Layers = bottomLayers
			middleLayers = append(middleLayers, layer)
		}
	}
	topLayer.Layers = middleLayers

	return *topLayer
}

func (v2Layer WMSLayer) MapToV3(v2Service WMSService) pdoknlv3.Layer {
	layer := pdoknlv3.Layer{
		Name:                &v2Layer.Name,
		Title:               v2Layer.Title,
		Abstract:            v2Layer.Abstract,
		Keywords:            v2Layer.Keywords,
		LabelNoClip:         v2Layer.LabelNoClip,
		Styles:              []pdoknlv3.Style{},
		Layers:              nil,
		BoundingBoxes:       []pdoknlv3.WMSBoundingBox{},
		MinScaleDenominator: nil,
		MaxScaleDenominator: nil,
		Visible:             smoothoperatorutils.PointerVal(v2Layer.Visible, true),
	}

	if v2Layer.SourceMetadataIdentifier != nil {
		layer.Authority = &pdoknlv3.Authority{
			Name:                     v2Service.Authority.Name,
			URL:                      v2Service.Authority.URL,
			SpatialDatasetIdentifier: *v2Layer.SourceMetadataIdentifier,
		}
	}

	if v2Layer.DatasetMetadataIdentifier != nil {
		layer.DatasetMetadataURL = &pdoknlv3.MetadataURL{
			CSW: &pdoknlv3.Metadata{
				MetadataIdentifier: *v2Layer.DatasetMetadataIdentifier,
			},
		}
	}

	if v2Layer.Extent != nil {
		layer.BoundingBoxes = append(layer.BoundingBoxes, pdoknlv3.WMSBoundingBox{
			CRS:  v2Service.DataEPSG,
			BBox: smoothoperatormodel.ExtentToBBox(*v2Layer.Extent),
		})
	} else if v2Service.Extent != nil {
		layer.BoundingBoxes = append(layer.BoundingBoxes, pdoknlv3.WMSBoundingBox{
			CRS:  v2Service.DataEPSG,
			BBox: smoothoperatormodel.ExtentToBBox(*v2Service.Extent),
		})
	}

	if v2Layer.MinScale != nil {
		layer.MinScaleDenominator = smoothoperatorutils.Pointer(strconv.FormatFloat(*v2Layer.MinScale, 'f', -1, 64))
	}

	if v2Layer.MaxScale != nil {
		layer.MaxScaleDenominator = smoothoperatorutils.Pointer(strconv.FormatFloat(*v2Layer.MaxScale, 'f', -1, 64))
	}

	for _, style := range v2Layer.Styles {
		v3Style := pdoknlv3.Style{
			Name:          style.Name,
			Title:         style.Title,
			Abstract:      style.Abstract,
			Visualization: style.Visualization,
		}

		if style.LegendFile != nil {
			v3Style.Legend = &pdoknlv3.Legend{
				BlobKey: style.LegendFile.BlobKey,
			}
		}

		layer.Styles = append(layer.Styles, v3Style)
	}

	if v2Layer.Data != nil {
		layer.Data = smoothoperatorutils.Pointer(ConvertV2DataToV3(*v2Layer.Data))
	}

	return layer
}

//nolint:cyclop
func mapV3LayerToV2Layers(v3Layer pdoknlv3.Layer, parent *pdoknlv3.Layer, serviceEPSG string) []WMSLayer {
	var layers []WMSLayer

	//nolint:nestif
	if parent == nil && v3Layer.Name == nil {
		// Default top layer, do not include in v2 layers
		if v3Layer.Layers != nil {
			for _, childLayer := range v3Layer.Layers {
				layers = append(layers, mapV3LayerToV2Layers(childLayer, nil, serviceEPSG)...)
			}
		}
	} else {
		v2Layer := WMSLayer{
			Name:        *v3Layer.Name,
			Title:       v3Layer.Title,
			Abstract:    v3Layer.Abstract,
			Keywords:    v3Layer.Keywords,
			LabelNoClip: v3Layer.LabelNoClip,
			Styles:      []Style{},
		}

		v2Layer.Visible = &v3Layer.Visible

		if parent != nil {
			v2Layer.Group = parent.Name
		}

		if v3Layer.DatasetMetadataURL != nil && v3Layer.DatasetMetadataURL.CSW != nil {
			v2Layer.DatasetMetadataIdentifier = &v3Layer.DatasetMetadataURL.CSW.MetadataIdentifier
		}

		if v3Layer.Authority != nil {
			v2Layer.SourceMetadataIdentifier = &v3Layer.Authority.SpatialDatasetIdentifier
		}

		for _, bb := range v3Layer.BoundingBoxes {
			if bb.CRS == serviceEPSG {
				v2Layer.Extent = smoothoperatorutils.Pointer(bb.BBox.ToExtent())
			}
		}

		if v3Layer.MinScaleDenominator != nil {
			val, err := strconv.ParseFloat(*v3Layer.MinScaleDenominator, 64)
			if err != nil {
				panic(err)
			}
			v2Layer.MinScale = &val
		}

		if v3Layer.MaxScaleDenominator != nil {
			val, err := strconv.ParseFloat(*v3Layer.MaxScaleDenominator, 64)
			if err != nil {
				panic(err)
			}
			v2Layer.MaxScale = &val
		}

		for _, v3Style := range v3Layer.Styles {
			v2Style := Style{
				Name:          v3Style.Name,
				Title:         v3Style.Title,
				Abstract:      v3Style.Abstract,
				Visualization: v3Style.Visualization,
			}

			if v3Style.Legend != nil {
				v2Style.LegendFile = &LegendFile{
					BlobKey: v3Style.Legend.BlobKey,
				}
			}

			v2Layer.Styles = append(v2Layer.Styles, v2Style)
		}

		if v3Layer.Data != nil {
			v2Layer.Data = smoothoperatorutils.Pointer(ConvertV3DataToV2(*v3Layer.Data))
		}

		layers = append(layers, v2Layer)

		if v3Layer.Layers != nil {
			for _, childLayer := range v3Layer.Layers {
				layers = append(layers, mapV3LayerToV2Layers(childLayer, &v3Layer, serviceEPSG)...)
			}
		}
	}

	return layers
}
