package capabilitiesgenerator

import "github.com/pdok/ogc-specifications/pkg/wms130"

// TODO Bounding boxes and default CRSes in this file are not used at the moment but are kept so we can use them again later

//nolint:unused
var defaultWMSBoundingBox = wms130.EXGeographicBoundingBox{
	WestBoundLongitude: 2.52713,
	EastBoundLongitude: 7.37403,
	SouthBoundLatitude: 50.2129,
	NorthBoundLatitude: 55.7212,
}

//nolint:unused
func getDefaultWMSCRSes() []wms130.CRS {
	return []wms130.CRS{{
		Namespace: "EPSG",
		Code:      28992,
	}, {
		Namespace: "EPSG",
		Code:      25831,
	}, {
		Namespace: "EPSG",
		Code:      25832,
	}, {
		Namespace: "EPSG",
		Code:      3034,
	}, {
		Namespace: "EPSG",
		Code:      3035,
	}, {
		Namespace: "EPSG",
		Code:      3857,
	}, {
		Namespace: "EPSG",
		Code:      4258,
	}, {
		Namespace: "EPSG",
		Code:      4326,
	}, {
		Namespace: "CRS",
		Code:      84,
	}}
}

//nolint:unused
func getDefaultWMSLayerBoundingBoxes() []*wms130.LayerBoundingBox {
	return []*wms130.LayerBoundingBox{
		{
			CRS:  "EPSG:28992",
			Minx: -25000,
			Miny: 250000,
			Maxx: 280000,
			Maxy: 860000,
			Resx: 0,
			Resy: 0,
		},
		{
			CRS:  "EPSG:25831",
			Minx: -470271,
			Miny: 5.56231e+06,
			Maxx: 795163,
			Maxy: 6.18197e+06,
			Resx: 0,
			Resy: 0,
		},
		{
			CRS:  "EPSG:25832",
			Minx: 62461.6,
			Miny: 5.56555e+06,
			Maxx: 397827,
			Maxy: 6.19042e+06,
			Resx: 0,
			Resy: 0,
		},
		{
			CRS:  "EPSG:3034",
			Minx: 2.61336e+06,
			Miny: 3.509e+06,
			Maxx: 3.22007e+06,
			Maxy: 3.84003e+06,
			Resx: 0,
			Resy: 0,
		},
		{
			CRS:  "EPSG:3035",
			Minx: 3.01676e+06,
			Miny: 3.81264e+06,
			Maxx: 3.64485e+06,
			Maxy: 4.15586e+06,
			Resx: 0,
			Resy: 0,
		},
		{
			CRS:  "EPSG:3857",
			Minx: 281318,
			Miny: 6.48322e+06,
			Maxx: 820873,
			Maxy: 7.50311e+06,
			Resx: 0,
			Resy: 0,
		},
		{
			CRS:  "EPSG:4258",
			Minx: 50.2129,
			Miny: 2.52713,
			Maxx: 55.7212,
			Maxy: 7.37403,
			Resx: 0,
			Resy: 0,
		},
		{
			CRS:  "EPSG:4326",
			Minx: 50.2129,
			Miny: 2.52713,
			Maxx: 55.7212,
			Maxy: 7.37403,
			Resx: 0,
			Resy: 0,
		},
		{
			CRS:  "CRS:84",
			Minx: 2.52713,
			Miny: 50.2129,
			Maxx: 7.37403,
			Maxy: 55.7212,
			Resx: 0,
			Resy: 0,
		},
	}
}
