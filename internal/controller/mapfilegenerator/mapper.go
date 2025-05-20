package mapfilegenerator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pdok/mapserver-operator/api/v2beta1"

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

var defaultEpsgList = []string{
	"EPSG:28992",
	"EPSG:25831",
	"EPSG:25832",
	"EPSG:3034",
	"EPSG:3035",
	"EPSG:3857",
	"EPSG:4258",
	"EPSG:4326",
}

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

	var extent string
	if wfs.Spec.Service.Bbox != nil {
		extent = wfs.Spec.Service.Bbox.DefaultCRS.ToExtent()
	}

	input := WFSInput{
		BaseServiceInput: BaseServiceInput{
			Title:           mapperutils.EscapeQuotes(wfs.Spec.Service.Title),
			Abstract:        mapperutils.EscapeQuotes(wfs.Spec.Service.Abstract),
			Keywords:        strings.Join(wfs.Spec.Service.KeywordsIncludingInspireKeyword(), ","),
			OnlineResource:  pdoknlv3.GetHost(true),
			Path:            "/" + pdoknlv3.GetBaseURLPath(wfs),
			MetadataID:      metadataID,
			Extent:          extent,
			NamespacePrefix: wfs.Spec.Service.Prefix,
			NamespaceURI:    mapperutils.GetNamespaceURI(wfs.Spec.Service.Prefix, ownerInfo),
			AutomaticCasing: wfs.Options().AutomaticCasing,
			DataEPSG:        wfs.Spec.Service.DefaultCrs,
			// TODO Should this be a constant like in v2, or OtherCRS + default
			EPSGList:   defaultEpsgList, // wfs.Spec.Service.OtherCrs,
			DebugLevel: mapserverDebugLevel,
		},
		MaxFeatures: smoothoperatorutils.PointerVal(wfs.Spec.Service.CountDefault, strconv.Itoa(defaultMaxFeatures)),
		Layers:      getWFSLayers(wfs.Spec.Service),
	}

	return input, nil
}

func getWFSLayers(service pdoknlv3.WFSService) (layers []WFSLayer) {
	for _, featureType := range service.FeatureTypes {
		layer := WFSLayer{
			BaseLayer: BaseLayer{
				Name:           featureType.Name,
				Title:          mapperutils.EscapeQuotes(featureType.Title),
				Abstract:       mapperutils.EscapeQuotes(featureType.Abstract),
				Keywords:       strings.Join(featureType.Keywords, ","),
				Extent:         getWFSExtent(featureType, service),
				MetadataID:     featureType.DatasetMetadataURL.CSW.MetadataIdentifier,
				Columns:        getColumns(featureType.Data),
				TableName:      featureType.Data.GetTableName(),
				GeometryType:   featureType.Data.GetGeometryType(),
				GeopackagePath: getGeopackagePath(featureType.Data),
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
	if featureType.Bbox != nil {
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

func getColumns(data pdoknlv3.Data) []Column {
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

func getGeopackagePath(data pdoknlv3.Data) *string {
	if data.Gpkg == nil {
		return nil
	}
	index := strings.LastIndex(data.Gpkg.BlobKey, "/") + 1
	blobName := data.Gpkg.BlobKey[index:]
	return smoothoperatorutils.Pointer(geopackagePath + "/" + blobName)
}

func MapWMSToMapfileGeneratorInput(wms *pdoknlv3.WMS, _ *smoothoperatorv1.OwnerInfo) (WMSInput, error) {
	service := wms.Spec.Service

	datasetOwner := ""
	if service.Layer.Authority != nil {
		datasetOwner = service.Layer.Authority.Name
	} else {
		datasetOwner = wms.ObjectMeta.Labels["dataset-owner"]
	}

	datasetName := wms.ObjectMeta.Labels["dataset"]
	protocol := "http"
	authority := wms.GetAuthority()
	authorityURL := ""
	if authority != nil {
		authorityURL = authority.URL
	}

	box := service.GetBoundingBox()
	extent := box.ToExtent()

	epsgs := []string{"EPSG:28992", "EPSG:25831", "EPSG:25832", "EPSG:3034", "EPSG:3035", "EPSG:3857", "EPSG:4258", "EPSG:4326", "CRS:84"}

	maxSize := "4000"
	if service.MaxSize != nil {
		maxSize = strconv.Itoa(int(*service.MaxSize))
	}

	var metadataID string
	if service.Inspire != nil {
		metadataID = service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier
	} else {
		metadataID = wms.ObjectMeta.Annotations[v2beta1.ServiceMetatdataIdentifierAnnotation]
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
			Title:           service.Title,
			Abstract:        service.Abstract,
			Keywords:        strings.Join(wms.Spec.Service.KeywordsIncludingInspireKeyword(), ","),
			Extent:          extent,
			NamespacePrefix: datasetName,
			NamespaceURI:    fmt.Sprintf("%s://%s.geonovum.nl", protocol, datasetName),
			OnlineResource:  pdoknlv3.GetHost(true),
			Path:            "/" + pdoknlv3.GetBaseURLPath(wms),
			MetadataID:      metadataID,
			DatasetOwner:    &datasetOwner,
			AuthorityURL:    &authorityURL,
			AutomaticCasing: wms.Options().AutomaticCasing,
			DataEPSG:        service.DataEPSG,
			EPSGList:        epsgs,
		},
		AccessConstraints: service.AccessConstraints,
		Layers:            []WMSLayer{},
		GroupLayers:       []GroupLayer{},
		Symbols:           getSymbols(wms),
		Fonts:             fonts,
		OutputFormatJpg:   "jpg",
		OutputFormatPng:   "png",
		Templates:         "/srv/data/config/templates",
		MaxSize:           maxSize,
	}

	annotatedLayers := wms.Spec.Service.GetAnnotatedLayers()
	for _, annotatedLayer := range annotatedLayers {
		if annotatedLayer.IsDataLayer {
			layer := getWMSLayer(annotatedLayer.Layer, extent, wms)
			result.Layers = append(result.Layers, layer)
		} else if annotatedLayer.IsGroupLayer && !annotatedLayer.IsTopLayer {
			groupLayer := GroupLayer{
				Name:       *annotatedLayer.Layer.Name,
				Title:      smoothoperatorutils.PointerVal(annotatedLayer.Layer.Title, ""),
				Abstract:   smoothoperatorutils.PointerVal(annotatedLayer.Layer.Abstract, ""),
				StyleName:  "",
				StyleTitle: "",
			}
			result.GroupLayers = append(result.GroupLayers, groupLayer)
		}
	}

	return result, nil
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
		columns = getColumns(*serviceLayer.Data)
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
		GroupName:                   groupName,
		Styles:                      []Style{},
		Offsite:                     "",
		GetFeatureInfoIncludesClass: false,
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
