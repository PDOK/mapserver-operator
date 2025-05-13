package ogcwebserviceproxy

import (
	"testing"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	controller "github.com/pdok/smooth-operator/pkg/util"
)

const expectedConfig = `grouplayers:
    grouplayer-1:
        - datalayer-1
        - datalayer-2
    grouplayer-2:
        - datalayer-3
        - datalayer-4
`

func TestGetConfig(t *testing.T) {
	type args struct {
		wms *pdoknlv3.WMS
	}
	tests := []struct {
		name       string
		args       args
		wantConfig string
		wantErr    bool
	}{
		{
			name: "GetConfig for OGC Webservice proxy",
			args: args{
				wms: &pdoknlv3.WMS{
					Spec: pdoknlv3.WMSSpec{
						Service: pdoknlv3.WMSService{
							Layer: pdoknlv3.Layer{
								Name: controller.Pointer("toplayer"),
								Layers: []pdoknlv3.Layer{
									{
										Name: controller.Pointer("grouplayer-1"),
										Layers: []pdoknlv3.Layer{
											{
												Name: controller.Pointer("datalayer-1"),
												Data: &pdoknlv3.Data{Gpkg: &pdoknlv3.Gpkg{BlobKey: "blob-1"}},
											},
											{
												Name: controller.Pointer("datalayer-2"),
												Data: &pdoknlv3.Data{Gpkg: &pdoknlv3.Gpkg{BlobKey: "blob-2"}},
											},
										},
									},
									{
										Name: controller.Pointer("grouplayer-2"),
										Layers: []pdoknlv3.Layer{
											{
												Name: controller.Pointer("datalayer-3"),
												Data: &pdoknlv3.Data{Gpkg: &pdoknlv3.Gpkg{BlobKey: "blob-3"}},
											},
											{
												Name: controller.Pointer("datalayer-4"),
												Data: &pdoknlv3.Data{Gpkg: &pdoknlv3.Gpkg{BlobKey: "blob-4"}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantConfig: expectedConfig,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConfig, err := GetConfig(tt.args.wms)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotConfig != tt.wantConfig {
				t.Errorf("GetConfig() gotConfig = %v, want %v", gotConfig, tt.wantConfig)
			}
		})
	}
}
