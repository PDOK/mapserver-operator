package featureinfogenerator

import (
	"testing"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

const (
	featureInfoGeneratorInput = `{
    "automaticCasing": true,
    "version": 2,
    "layers": [
        {
            "name": "gpkg-layer-name",
            "groupName": "group-layer-name",
            "properties": [
                {
                    "name": "fuuid"
                },
                {
                    "name": "column-1",
                    "alias": "ALIAS_column-1"
                },
                {
                    "name": "column-2"
                }
            ]
        },
        {
            "name": "postgis-layer-name",
            "groupName": "group-layer-name",
            "properties": [
                {
                    "name": "fuuid"
                },
                {
                    "name": "column-1"
                },
                {
                    "name": "column-2"
                }
            ]
        },
        {
            "name": "tif-layer-name",
            "groupName": "group-layer-name",
            "properties": [
                {
                    "name": "value_list"
                },
                {
                    "name": "class"
                }
            ]
        }
    ]
}`
)

func TestGetInput(t *testing.T) {
	type args struct {
		wms *pdoknlv3.WMS
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "GetInput for featureinfoGenerator",
			args: args{
				wms: &pdoknlv3.WMS{
					Spec: pdoknlv3.WMSSpec{
						Options: pdoknlv3.Options{
							AutomaticCasing: true,
						},
						Service: pdoknlv3.WMSService{
							Layer: pdoknlv3.Layer{
								Name: smoothoperatorutils.Pointer("top-layer-name"),
								Layers: []pdoknlv3.Layer{
									{
										Name: smoothoperatorutils.Pointer("group-layer-name"),
										Layers: []pdoknlv3.Layer{
											{
												Name: smoothoperatorutils.Pointer("gpkg-layer-name"),
												Data: &pdoknlv3.Data{
													Gpkg: &pdoknlv3.Gpkg{
														Columns: []pdoknlv3.Column{
															{Name: "column-1", Alias: smoothoperatorutils.Pointer("ALIAS_column-1")},
															{Name: "column-2"},
														},
													},
												},
											},
											{
												Name: smoothoperatorutils.Pointer("postgis-layer-name"),
												Data: &pdoknlv3.Data{
													Postgis: &pdoknlv3.Postgis{
														Columns: []pdoknlv3.Column{
															{Name: "column-1"},
															{Name: "column-2"},
														},
													},
												},
											},
											{
												Name: smoothoperatorutils.Pointer("tif-layer-name"),
												Data: &pdoknlv3.Data{
													TIF: &pdoknlv3.TIF{
														GetFeatureInfoIncludesClass: true,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want:    featureInfoGeneratorInput,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetInput(tt.args.wms)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetInput() got = %v, want %v", got, tt.want)
			}
		})
	}
}
