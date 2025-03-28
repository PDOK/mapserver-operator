package mapfilegenerator

import (
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	shared_model "github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
	WFSConfig = `{
    "service_title": "some Service title",
    "service_abstract": "some \\\"Service\\\" abstract",
    "service_keywords": "service-keyword-1,service-keyword-2,infoFeatureAccessService",
    "service_accessconstraints": "http://creativecommons.org/publicdomain/zero/1.0/deed.nl",
    "service_extent": "0.0 2.0 1.0 3.0",
    "service_wfs_maxfeatures": "1000",
    "service_namespace_prefix": "prefix",
    "service_namespace_uri": "http://prefix.geonovum.nl",
    "service_onlineresource": "http://localhost",
    "service_path": "/datasetOwner/dataset/theme/wfs/v1_0",
    "service_metadata_id": "metameta-meta-meta-meta-metametameta",
    "automatic_casing": true,
    "data_epsg": "EPSG:28992",
    "epsg_list": [
        "EPSG:28992",
        "EPSG:25831",
        "EPSG:25832",
        "EPSG:3034",
        "EPSG:3035",
        "EPSG:3857",
        "EPSG:4258",
        "EPSG:4326"
    ],
    "layers": [
        {
            "name": "featuretype-1-name",
            "title": "featuretype-1-title",
            "abstract": "feature \\\"1\\\" abstract",
            "keywords": "featuretype-1-keyword-1,featuretype-1-keyword-2",
            "layer_extent": "0.0 2.0 1.0 3.0",
            "dataset_metadata_id": "datadata-data-data-data-datadatadata",
            "columns": [
                {
                    "name": "fuuid"
                },
                {
                    "name": "featuretype-1-column-1"
                },
                {
                    "name": "featuretype-1-column-2"
                }
            ],
            "geometry_type": "Point",
            "gpkg_path": "/srv/data/gpkg/file-1.gpkg",
            "tablename": "featuretype-1"
        },
        {
            "name": "featuretype-2-name",
            "title": "featuretype-2-title",
            "abstract": "feature \\\"2\\\" abstract",
            "keywords": "featuretype-2-keyword-1,featuretype-2-keyword-2",
            "layer_extent": "0.0 2.0 1.0 3.0",
            "dataset_metadata_id": "datadata-data-data-data-datadatadata",
            "columns": [
                {
                    "name": "fuuid"
                },
                {
                    "name": "featuretype-2-column-1",
                    "alias": "alias_featuretype-2-column-1"
                },
                {
                    "name": "featuretype-2-column-2"
                }
            ],
            "geometry_type": "MultiLine",
            "tablename": "featuretype-2",
            "postgis": true
        }
    ]
}`
)

func TestGetConfigForWFS(t *testing.T) {
	type args struct {
		WFS       *pdoknlv3.WFS
		ownerInfo *smoothoperatorv1.OwnerInfo
	}
	pdoknlv3.SetHost("http://localhost")
	tests := []struct {
		name       string
		args       args
		wantConfig string
		wantErr    bool
	}{
		{
			name: "GetConfig for WFS",
			args: args{
				WFS: &pdoknlv3.WFS{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"dataset":         "dataset",
							"dataset-owner":   "datasetOwner",
							"theme":           "theme",
							"service-version": "v1_0",
						},
					},
					Spec: pdoknlv3.WFSSpec{
						Options: &pdoknlv3.Options{
							AutomaticCasing: true,
						},
						Service: pdoknlv3.WFSService{
							Title:             "some Service title",
							Abstract:          "some \"Service\" abstract",
							Keywords:          []string{"service-keyword-1", "service-keyword-2", "infoFeatureAccessService"},
							AccessConstraints: "http://creativecommons.org/publicdomain/zero/1.0/deed.nl",
							Bbox: &pdoknlv3.Bbox{
								DefaultCRS: shared_model.BBox{
									MinX: "0.0",
									MaxX: "1.0",
									MinY: "2.0",
									MaxY: "3.0",
								},
							},
							Inspire: &pdoknlv3.Inspire{
								ServiceMetadataURL: pdoknlv3.MetadataURL{
									CSW: &pdoknlv3.Metadata{
										MetadataIdentifier: "metameta-meta-meta-meta-metametameta",
									},
								},
							},
							DefaultCrs: "EPSG:28992",
							OtherCrs: []string{
								"EPSG:28992",
								"EPSG:25831",
								"EPSG:25832",
								"EPSG:3034",
								"EPSG:3035",
								"EPSG:3857",
								"EPSG:4258",
								"EPSG:4326",
							},
							FeatureTypes: []pdoknlv3.FeatureType{
								{
									Name:     "featuretype-1-name",
									Title:    "featuretype-1-title",
									Abstract: "feature \"1\" abstract",
									Keywords: []string{"featuretype-1-keyword-1", "featuretype-1-keyword-2"},
									DatasetMetadataURL: pdoknlv3.MetadataURL{
										CSW: &pdoknlv3.Metadata{
											MetadataIdentifier: "datadata-data-data-data-datadatadata",
										},
									},
									Bbox: &pdoknlv3.FeatureBbox{
										DefaultCRS: shared_model.BBox{
											MinX: "0.0",
											MaxX: "1.0",
											MinY: "2.0",
											MaxY: "3.0",
										},
									},
									Data: pdoknlv3.Data{
										Gpkg: &pdoknlv3.Gpkg{
											TableName:    "featuretype-1",
											GeometryType: "Point",
											BlobKey:      "public/testme/gpkg/file-1.gpkg",
											Columns: []pdoknlv3.Column{
												{Name: "featuretype-1-column-1"},
												{Name: "featuretype-1-column-2"},
											},
										},
									},
								},
								{
									Name:     "featuretype-2-name",
									Title:    "featuretype-2-title",
									Abstract: "feature \"2\" abstract",
									Keywords: []string{"featuretype-2-keyword-1", "featuretype-2-keyword-2"},
									DatasetMetadataURL: pdoknlv3.MetadataURL{
										CSW: &pdoknlv3.Metadata{
											MetadataIdentifier: "datadata-data-data-data-datadatadata",
										},
									},
									Bbox: &pdoknlv3.FeatureBbox{
										DefaultCRS: shared_model.BBox{
											MinX: "0.0",
											MaxX: "1.0",
											MinY: "2.0",
											MaxY: "3.0",
										},
									},
									Data: pdoknlv3.Data{
										Postgis: &pdoknlv3.Postgis{
											TableName:    "featuretype-2",
											GeometryType: "MultiLine",
											Columns: []pdoknlv3.Column{
												{Name: "featuretype-2-column-1", Alias: smoothoperatorutils.Pointer("alias_featuretype-2-column-1")},
												{Name: "featuretype-2-column-2"},
											},
										},
									},
								},
							},
							Prefix: "prefix",
						},
					},
				},
				ownerInfo: &smoothoperatorv1.OwnerInfo{
					Spec: smoothoperatorv1.OwnerInfoSpec{
						NamespaceTemplate: "http://{{prefix}}.geonovum.nl",
					},
				},
			},
			wantConfig: WFSConfig,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConfig, err := GetConfig(tt.args.WFS, tt.args.ownerInfo)
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
