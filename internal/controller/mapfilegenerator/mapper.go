package mapfilegenerator

import (
	"slices"
	"strconv"
	"strings"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

const (
	defaultMaxFeatures = 1000
	tifPath            = "/srv/data/tif"
	geopackagePath     = "/srv/data/gpkg"
	defaultExtent      = "-25000 250000 280000 860000"
)

var mapserverDebugLevel = 0

func SetDebugLevel(level int) {
	if level < 0 || level > 5 {
		panic("level must be between 0 and 5")
	}

	mapserverDebugLevel = level
}

func MapWFSToMapfileGeneratorInput(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (WFSInput, error) {
	var metadataID string
	if wfs.Spec.Service.Inspire != nil {
		metadataID = wfs.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier
	}

	extent := defaultExtent
	if wfs.Spec.Service.Bbox != nil {
		extent = wfs.Spec.Service.Bbox.DefaultCRS.ToExtent()
	}

	input := WFSInput{
		BaseServiceInput: BaseServiceInput{
			Title:             wfs.Spec.Service.Title,
			Abstract:          wfs.Spec.Service.Abstract,
			Keywords:          strings.Join(wfs.Spec.Service.KeywordsIncludingInspireKeyword(), ","),
			OnlineResource:    wfs.URL().Scheme + "://" + wfs.URL().Host,
			Path:              wfs.URL().Path,
			MetadataID:        metadataID,
			Extent:            extent,
			NamespacePrefix:   wfs.Spec.Service.Prefix,
			NamespaceURI:      mapperutils.GetNamespaceURI(wfs.Spec.Service.Prefix, ownerInfo),
			AutomaticCasing:   wfs.Options().AutomaticCasing,
			DataEPSG:          wfs.Spec.Service.DefaultCrs,
			EPSGList:          append([]string{wfs.Spec.Service.DefaultCrs}, wfs.Spec.Service.OtherCrs...),
			DebugLevel:        mapserverDebugLevel,
			AccessConstraints: wfs.Spec.Service.AccessConstraints.String(),
		},
		MaxFeatures: strconv.Itoa(smoothoperatorutils.PointerVal(wfs.Spec.Service.CountDefault, defaultMaxFeatures)),
		Layers:      getWFSLayers(wfs.Spec.Service),
	}

	return input, nil
}

func getWFSLayers(service pdoknlv3.WFSService) (layers []WFSLayer) {
	for _, featureType := range service.FeatureTypes {
		layer := WFSLayer{
			BaseLayer: BaseLayer{
				Name:           featureType.Name,
				Title:          featureType.Title,
				Abstract:       featureType.Abstract,
				Keywords:       strings.Join(featureType.Keywords, ","),
				Extent:         getWFSExtent(featureType, service),
				MetadataID:     featureType.DatasetMetadataURL.CSW.MetadataIdentifier,
				Columns:        getColumns(featureType.Data),
				TableName:      featureType.Data.GetTableName(),
				GeometryType:   featureType.Data.GetGeometryType(),
				GeopackagePath: getGeopackagePath(featureType.Data.Gpkg),
			},
		}
		if featureType.Data.Postgis != nil {
			layer.Postgis = smoothoperatorutils.Pointer(true)
		}

		layers = append(layers, layer)
	}

	return
}

func getWFSExtent(featureType pdoknlv3.FeatureType, service pdoknlv3.WFSService) string {
	if featureType.Bbox != nil && featureType.Bbox.DefaultCRS != nil {
		return featureType.Bbox.DefaultCRS.ToExtent()
	}
	if service.Bbox != nil {
		return service.Bbox.DefaultCRS.ToExtent()
	}
	return defaultExtent
}

func getWMSExtent(serviceLayer pdoknlv3.Layer, serviceExtent string) string {
	if len(serviceLayer.BoundingBoxes) > 0 {
		return serviceLayer.BoundingBoxes[0].ToExtent()
	}
	if serviceExtent != "" {
		return serviceExtent
	}
	return defaultExtent
}

func getColumns(data pdoknlv3.BaseData) []Column {
	columns := []Column{{Name: "fuuid"}}
	if data.GetColumns() != nil {
		for _, column := range *data.GetColumns() {
			columns = append(columns, Column{Name: column.Name, Alias: column.Alias})
		}
	} else {
		return nil
	}
	return columns
}

func getGeopackagePath(gpkg *pdoknlv3.Gpkg) *string {
	if gpkg == nil {
		return nil
	}
	index := strings.LastIndex(gpkg.BlobKey, "/") + 1
	blobName := gpkg.BlobKey[index:]
	return smoothoperatorutils.Pointer(geopackagePath + "/" + blobName)
}

func MapWMSToMapfileGeneratorInput(wms *pdoknlv3.WMS, ownerInfo *smoothoperatorv1.OwnerInfo) (WMSInput, error) {
	service := wms.Spec.Service

	authority := wms.GetAuthority()
	authorityURL := ""
	datasetOwner := ""
	if authority != nil {
		authorityURL = authority.URL
		datasetOwner = authority.Name
	}

	box := service.GetBoundingBox()
	extent := box.ToExtent()

	maxSize := "4000"
	if service.MaxSize != nil {
		maxSize = strconv.Itoa(int(*service.MaxSize))
	}

	metadataID := ""
	if service.Inspire != nil {
		metadataID = service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier
	}

	var fonts *string

	if service.StylingAssets != nil {
		writeFonts := mapperutils.AnyMatch(service.StylingAssets.BlobKeys, func(s string) bool {
			return strings.HasSuffix(s, ".ttf")
		})

		if writeFonts {
			fonts = smoothoperatorutils.Pointer("/srv/data/config/fonts")
		}
	}

	result := WMSInput{
		BaseServiceInput: BaseServiceInput{
			Title:             service.Title,
			Abstract:          service.Abstract,
			Keywords:          strings.Join(wms.Spec.Service.KeywordsIncludingInspireKeyword(), ","),
			Extent:            extent,
			NamespacePrefix:   wms.Spec.Service.Prefix,
			NamespaceURI:      mapperutils.GetNamespaceURI(wms.Spec.Service.Prefix, ownerInfo),
			OnlineResource:    wms.URL().Scheme + "://" + wms.URL().Host,
			Path:              wms.URL().Path,
			MetadataID:        metadataID,
			DatasetOwner:      &datasetOwner,
			AuthorityURL:      &authorityURL,
			AutomaticCasing:   wms.Options().AutomaticCasing,
			DataEPSG:          service.DataEPSG,
			AccessConstraints: service.AccessConstraints.String(),
		},
		Layers:          []WMSLayer{},
		GroupLayers:     []GroupLayer{},
		Symbols:         getSymbols(wms),
		Fonts:           fonts,
		OutputFormatJpg: "jpg",
		OutputFormatPng: "png",
		Templates:       constants.HTMLTemplatesPath,
		MaxSize:         maxSize,
	}

	if wms.Spec.Service.Layer.Name != nil {
		result.TopLevelName = *wms.Spec.Service.Layer.Name
	}

	if wms.Spec.Service.Resolution != nil {
		result.Resolution = strconv.Itoa(int(*wms.Spec.Service.Resolution))
	}
	if wms.Spec.Service.DefResolution != nil {
		result.DefResolution = strconv.Itoa(int(*wms.Spec.Service.DefResolution))
	}

	mapLayers(wms, extent, &result)

	return result, nil
}

func mapLayers(wms *pdoknlv3.WMS, extent string, result *WMSInput) {
	epsgs := []string{}

	annotatedLayers := wms.Spec.Service.GetAnnotatedLayers()
	for _, annotatedLayer := range annotatedLayers {

		for _, bbox := range annotatedLayer.BoundingBoxes {
			if !slices.Contains(epsgs, bbox.CRS) {
				epsgs = append(epsgs, bbox.CRS)
			}
		}

		if annotatedLayer.IsDataLayer {
			layer := getWMSLayer(annotatedLayer.Layer, extent, wms)
			result.Layers = append(result.Layers, layer)
		} else if annotatedLayer.IsGroupLayer && !annotatedLayer.IsTopLayer {
			groupLayer := GroupLayer{
				Name:       *annotatedLayer.Name,
				Title:      smoothoperatorutils.PointerVal(annotatedLayer.Title, ""),
				Abstract:   smoothoperatorutils.PointerVal(annotatedLayer.Abstract, ""),
				StyleName:  "",
				StyleTitle: "",
			}
			if len(annotatedLayer.Styles) > 0 {
				groupLayer.StyleName = annotatedLayer.Layer.Styles[0].Name
				groupLayer.StyleTitle = smoothoperatorutils.PointerVal(annotatedLayer.Layer.Styles[0].Title, "")
			}
			result.GroupLayers = append(result.GroupLayers, groupLayer)
		}
	}

	if !slices.Contains(epsgs, "CRS:84") {
		epsgs = append(epsgs, "CRS:84")
	}

	result.EPSGList = epsgs
}

func getWMSLayer(serviceLayer pdoknlv3.Layer, serviceExtent string, wms *pdoknlv3.WMS) WMSLayer {
	groupName := ""
	parent := wms.Spec.Service.GetParentLayer(serviceLayer)
	// If the layer falls directly under the toplayer, the groupname is omitted
	if !parent.IsTopLayer() && parent.IsGroupLayer() && parent.Name != nil && parent.Visible {
		groupName = *parent.Name
	}

	var columns []Column
	if serviceLayer.Data != nil {
		columns = getColumns(serviceLayer.Data.BaseData)
	}

	var tableName *string
	if serviceLayer.Data != nil {
		tableName = serviceLayer.Data.GetTableName()
	}

	metadataID := ""
	if serviceLayer.DatasetMetadataURL != nil && serviceLayer.DatasetMetadataURL.CSW != nil {
		metadataID = serviceLayer.DatasetMetadataURL.CSW.MetadataIdentifier
	}

	result := WMSLayer{
		BaseLayer: BaseLayer{
			Name:           *serviceLayer.Name,
			Title:          smoothoperatorutils.PointerVal(serviceLayer.Title, ""),
			Abstract:       smoothoperatorutils.PointerVal(serviceLayer.Abstract, ""),
			Keywords:       strings.Join(serviceLayer.Keywords, ","),
			Extent:         getWMSExtent(serviceLayer, serviceExtent),
			MetadataID:     metadataID,
			Columns:        columns,
			GeometryType:   nil,
			GeopackagePath: nil,
			TableName:      tableName,
			Postgis:        nil,
			MinScale:       serviceLayer.MinScaleDenominator,
			MaxScale:       serviceLayer.MaxScaleDenominator,
			LabelNoClip:    serviceLayer.LabelNoClip,
		},
		GroupName: groupName,
		Styles:    []Style{},
		Offsite:   "",
	}

	for _, style := range serviceLayer.Styles {
		stylePath := "/styling/" + smoothoperatorutils.PointerVal(style.Visualization, "")
		result.Styles = append(result.Styles, Style{
			Path:  stylePath,
			Title: smoothoperatorutils.PointerVal(style.Title, ""),
		})
	}

	if serviceLayer.Data != nil {
		SetDataFields(wms, &result, *serviceLayer.Data)
	}

	return result
}

func getSymbols(wms *pdoknlv3.WMS) []string {
	result := make([]string, 0)
	service := wms.Spec.Service
	if service.StylingAssets != nil {
		for _, ref := range service.StylingAssets.ConfigMapRefs {
			for _, key := range ref.Keys {
				if strings.HasSuffix(key, ".symbol") {
					result = append(result, "/styling/"+key)
				}
			}
		}
	}
	return result
}
