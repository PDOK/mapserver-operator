package mapfilegenerator

import (
	"fmt"
	"strconv"
	"strings"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

const (
	defaultMaxFeatures = 1000
	geopackagePath     = "/srv/data/gpkg"
)

func MapWFSToMapfileGeneratorInput(wfs *pdoknlv3.WFS, ownerInfo *smoothoperatorv1.OwnerInfo) (WFSInput, error) {
	input := WFSInput{
		BaseServiceInput: BaseServiceInput{
			Title:           mapperutils.EscapeQuotes(wfs.Spec.Service.Title),
			Abstract:        mapperutils.EscapeQuotes(wfs.Spec.Service.Abstract),
			Keywords:        strings.Join(wfs.Spec.Service.Keywords, ","),
			OnlineResource:  pdoknlv3.GetHost(),
			Path:            mapperutils.GetPath(wfs),
			MetadataId:      wfs.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier,
			Extent:          wfs.Spec.Service.Bbox.DefaultCRS.ToExtent(),
			NamespacePrefix: wfs.Spec.Service.Prefix,
			NamespaceURI:    mapperutils.GetNamespaceURI(wfs.Spec.Service.Prefix, ownerInfo),
			AutomaticCasing: wfs.Spec.Options.AutomaticCasing,
			DataEPSG:        wfs.Spec.Service.DefaultCrs,
			// TODO Should this be a constant like in v2, or OtherCRS + default
			EPSGList: wfs.Spec.Service.OtherCrs,
		},
		MaxFeatures: smoothoperatorutils.PointerVal(wfs.Spec.Service.CountDefault, strconv.Itoa(defaultMaxFeatures)),
		Layers:      getWFSLayers(wfs.Spec.Service.FeatureTypes),
	}

	return input, nil
}

func getWFSLayers(featureTypes []pdoknlv3.FeatureType) (layers []WFSLayer) {
	for _, featureType := range featureTypes {
		bbox := pdoknlv3.FeatureBbox{}
		if featureType.Bbox != nil {
			bbox = *featureType.Bbox
		}
		layer := WFSLayer{
			BaseLayer: BaseLayer{
				Name:           featureType.Name,
				Title:          mapperutils.EscapeQuotes(featureType.Title),
				Abstract:       mapperutils.EscapeQuotes(featureType.Abstract),
				Keywords:       strings.Join(featureType.Keywords, ","),
				Extent:         bbox.DefaultCRS.ToExtent(),
				MetadataId:     featureType.DatasetMetadataURL.CSW.MetadataIdentifier,
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

func getColumns(data pdoknlv3.Data) []Column {
	columns := []Column{{Name: "fuuid"}}
	for _, column := range *data.GetColumns() {
		columns = append(columns, Column{Name: column.Name, Alias: column.Alias})
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

func MapWMSToMapfileGeneratorInput(wms *pdoknlv3.WMS, ownerInfo *smoothoperatorv1.OwnerInfo) (WMSInput, error) {
	service := wms.Spec.Service

	accessConstraints := service.AccessConstraints
	if accessConstraints == "" {
		accessConstraints = "https://creativecommons.org/publicdomain/zero/1.0/deed.nl"
	}

	owner := wms.ObjectMeta.Labels["dataset-owner"]
	datasetName := wms.ObjectMeta.Labels["dataset"]
	protocol := "http"
	authority := wms.GetAuthority()
	authorityUrl := ""
	if authority != nil {
		authorityUrl = authority.URL
	}

	box := service.GetBoundingBox()
	extent := box.ToExtent()

	epsgs := []string{"EPSG:28992", "EPSG:25831", "EPSG:25832", "EPSG:3034", "EPSG:3035", "EPSG:3857", "EPSG:4258", "EPSG:4326", "CRS:84"}

	result := WMSInput{
		BaseServiceInput: BaseServiceInput{
			Title:           service.Title,
			Abstract:        service.Abstract,
			Keywords:        strings.Join(service.Keywords, ","),
			Extent:          extent,
			NamespacePrefix: datasetName,
			NamespaceURI:    fmt.Sprintf("%s://%s.geonovum.nl", protocol, datasetName),
			OnlineResource:  pdoknlv3.GetHost(),
			Path:            mapperutils.GetPath(wms),
			MetadataId:      wms.Spec.Service.Inspire.ServiceMetadataURL.CSW.MetadataIdentifier,
			DatasetOwner:    &owner,
			AuthorityURL:    &authorityUrl,
			AutomaticCasing: wms.Spec.Options.AutomaticCasing,
			DataEPSG:        service.DataEPSG,
			EPSGList:        epsgs,
		},
		AccessConstraints: accessConstraints,
		Layers:            []WMSLayer{},
		Templates:         "/srv/data/config/templates",
	}

	allLayers := wms.Spec.Service.GetAllLayers()
	for _, serviceLayer := range allLayers[1:] {
		layer := getWMSLayer(serviceLayer, extent, wms)
		result.Layers = append(result.Layers, layer)
	}
	return result, nil
}

func getWMSLayer(serviceLayer pdoknlv3.Layer, serviceExtent string, wms *pdoknlv3.WMS) WMSLayer {
	layerExtent := serviceExtent
	if len(serviceLayer.BoundingBoxes) > 0 {
		layerExtent = serviceLayer.BoundingBoxes[0].ToExtent()
	}

	result := WMSLayer{
		BaseLayer: BaseLayer{
			Name:           serviceLayer.Name,
			Title:          smoothoperatorutils.PointerVal(serviceLayer.Title, ""),
			Abstract:       smoothoperatorutils.PointerVal(serviceLayer.Abstract, ""),
			Keywords:       strings.Join(serviceLayer.Keywords, ","),
			Extent:         layerExtent,
			MetadataId:     serviceLayer.DatasetMetadataURL.CSW.MetadataIdentifier,
			Columns:        getColumns(*serviceLayer.Data),
			GeometryType:   nil,
			GeopackagePath: nil,
			TableName:      serviceLayer.Data.GetTableName(),
			Postgis:        nil,
		},
		GroupName:                   "",
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
		if serviceLayer.Data.Gpkg != nil {
			gpkg := serviceLayer.Data.Gpkg

			result.GeometryType = &gpkg.GeometryType
			geopackageConstructedPath := ""
			if smoothoperatorutils.PointerVal(wms.Options().PrefetchData, true) {
				splitBlobKey := strings.Split(gpkg.BlobKey, "/")
				geopackageConstructedPath = "/srv/data/gpkg/" + splitBlobKey[len(splitBlobKey)-1]
			} else {
				geopackageConstructedPath = "/vsiaz/geopackages/" + gpkg.BlobKey
			}

			result.GeopackagePath = &geopackageConstructedPath
		} else if serviceLayer.Data.TIF != nil {
			tif := serviceLayer.Data.TIF
			result.GeometryType = smoothoperatorutils.Pointer("Raster")
			_ = tif

		} else if serviceLayer.Data.Postgis != nil {
			postgis := serviceLayer.Data.Postgis
			_ = postgis
		}
	}

	return result
}
