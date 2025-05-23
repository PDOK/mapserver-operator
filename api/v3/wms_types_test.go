package v3

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

func TestLayer_setInheritedBoundingBoxes(t *testing.T) {
	first28992BoundingBox := WMSBoundingBox{
		CRS: "EPSG:28992",
		BBox: smoothoperatormodel.BBox{
			MinX: "482.06",
			MaxX: "306602.42",
			MinY: "284182.97",
			MaxY: "637049.52",
		},
	}
	first4326BoundingBox := WMSBoundingBox{
		CRS: "EPSG:4326",
		BBox: smoothoperatormodel.BBox{
			MinX: "2.35417303",
			MaxX: "7.5553525",
			MinY: "50.71447164",
			MaxY: "55.66948102",
		},
	}
	first4258BoundingBox := WMSBoundingBox{
		CRS: "EPSG:4258",
		BBox: smoothoperatormodel.BBox{
			MinX: "2.354173",
			MaxX: "7.5553527",
			MinY: "50.71447",
			MaxY: "55.66948",
		}}
	second28992BoundingBox := WMSBoundingBox{
		CRS: "EPSG:28992",
		BBox: smoothoperatormodel.BBox{
			MinX: "0.00",
			MaxX: "310000.00",
			MinY: "275000.00",
			MaxY: "650000.00",
		}}

	tests := []struct {
		name                                string
		layer                               Layer
		toplayerExpectedBoundingBoxCount    int
		toplayerExpectedBoundingBoxes       []WMSBoundingBox
		grouplayer1ExpectedBoundingBoxCount int
		grouplayer1ExpectedBoundingBoxes    []WMSBoundingBox
		datalayer1ExpectedBoundingBoxCount  int
		datalayer1ExpectedBoundingBoxes     []WMSBoundingBox
		datalayer2ExpectedBoundingBoxCount  int
		datalayer2ExpectedBoundingBoxes     []WMSBoundingBox
	}{
		{
			name: "setInheritedBoundingBoxes for layer",
			layer: Layer{
				Name:          smoothoperatorutils.Pointer("toplayer"),
				BoundingBoxes: []WMSBoundingBox{first28992BoundingBox},
				Layers: []Layer{
					{
						Name:          smoothoperatorutils.Pointer("grouplayer-1"),
						BoundingBoxes: []WMSBoundingBox{first4326BoundingBox},
						Layers: []Layer{
							{
								Name:          smoothoperatorutils.Pointer("datalayer-1"),
								BoundingBoxes: []WMSBoundingBox{first4258BoundingBox},
							},
							{
								Name:          smoothoperatorutils.Pointer("datalayer-2"),
								BoundingBoxes: []WMSBoundingBox{second28992BoundingBox},
							},
						},
					},
				},
			},
			toplayerExpectedBoundingBoxCount:    1,
			toplayerExpectedBoundingBoxes:       []WMSBoundingBox{first28992BoundingBox},
			grouplayer1ExpectedBoundingBoxCount: 2,
			grouplayer1ExpectedBoundingBoxes:    []WMSBoundingBox{first4326BoundingBox, first28992BoundingBox},
			datalayer1ExpectedBoundingBoxCount:  3,
			datalayer1ExpectedBoundingBoxes:     []WMSBoundingBox{first4258BoundingBox, first4326BoundingBox, first28992BoundingBox},
			datalayer2ExpectedBoundingBoxCount:  2,
			datalayer2ExpectedBoundingBoxes:     []WMSBoundingBox{second28992BoundingBox, first4326BoundingBox},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layer := tt.layer
			layer.setInheritedBoundingBoxes()

			topChildLayers := layer.Layers
			groupLayer1 := topChildLayers[0]
			groupChildLayers := groupLayer1.Layers
			dataLayer1 := groupChildLayers[0]
			dataLayer2 := groupChildLayers[1]

			if len(layer.BoundingBoxes) != tt.toplayerExpectedBoundingBoxCount {
				t.Errorf("Toplayer has unexpected number of bounding boxes = %v, want %v", len(layer.BoundingBoxes), tt.toplayerExpectedBoundingBoxCount)
			}
			if !cmp.Equal(layer.BoundingBoxes, tt.toplayerExpectedBoundingBoxes) {
				t.Errorf("Toplayer has unexpected bounding boxes = %v, want %v", layer.BoundingBoxes, tt.toplayerExpectedBoundingBoxes)
			}
			if len(groupLayer1.BoundingBoxes) != tt.grouplayer1ExpectedBoundingBoxCount {
				t.Errorf("Grouplayer has unexpected number of bounding boxes = %v, want %v", len(groupLayer1.BoundingBoxes), tt.grouplayer1ExpectedBoundingBoxCount)
			}
			if !cmp.Equal(groupLayer1.BoundingBoxes, tt.grouplayer1ExpectedBoundingBoxes) {
				t.Errorf("Grouplayer has unexpected bounding boxes = %v, want %v", groupLayer1.BoundingBoxes, tt.grouplayer1ExpectedBoundingBoxes)
			}
			if len(dataLayer1.BoundingBoxes) != tt.datalayer1ExpectedBoundingBoxCount {
				t.Errorf("Datalayer1 has unexpected number of bounding boxes = %v, want %v", len(dataLayer1.BoundingBoxes), tt.datalayer1ExpectedBoundingBoxCount)
			}
			if !cmp.Equal(dataLayer1.BoundingBoxes, tt.datalayer1ExpectedBoundingBoxes) {
				t.Errorf("Datalayer1 has unexpected bounding boxes = %v, want %v", dataLayer1.BoundingBoxes, tt.datalayer1ExpectedBoundingBoxes)
			}
			if len(dataLayer2.BoundingBoxes) != tt.datalayer2ExpectedBoundingBoxCount {
				t.Errorf("Datalayer2 has unexpected number of bounding boxes = %v, want %v", len(dataLayer2.BoundingBoxes), tt.datalayer2ExpectedBoundingBoxCount)
			}
			if !cmp.Equal(dataLayer2.BoundingBoxes, tt.datalayer2ExpectedBoundingBoxes) {
				t.Errorf("Datalayer2 has unexpected bounding boxes = %v, want %v", dataLayer2.BoundingBoxes, tt.datalayer2ExpectedBoundingBoxes)
			}
		})
	}
}

func TestLayer_GetParent(t *testing.T) {
	childLayer2 := Layer{Name: smoothoperatorutils.Pointer("childlayer-2")}
	childLayer1 := Layer{Name: smoothoperatorutils.Pointer("childlayer-1"), Layers: []Layer{childLayer2}}
	topLayer := Layer{Name: smoothoperatorutils.Pointer("toplayer"), Layers: []Layer{childLayer1}}

	type args struct {
		service WMSService
	}
	tests := []struct {
		name  string
		layer Layer
		args  args
		want  *Layer
	}{
		{
			name:  "Test GetParent on layer with parent",
			layer: childLayer2,
			args:  args{service: WMSService{Layer: topLayer}},
			want:  &childLayer1,
		},
		{
			name:  "Test GetParent on layer without parent",
			layer: topLayer,
			args:  args{service: WMSService{Layer: topLayer}},
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.service.GetParentLayer(tt.layer); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetParent() = %v, want %v", got, tt.want)
			}
		})
	}
}
