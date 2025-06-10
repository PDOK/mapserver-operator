package blobdownload

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
)

const (
	WFSArgsWithPrefetch = `set -e;
mkdir -p /srv/data/config/;
rclone config create --non-interactive --obscure blobs azureblob endpoint $BLOBS_ENDPOINT account $BLOBS_ACCOUNT key $BLOBS_KEY use_emulator true;
bash /srv/scripts/gpkg_download.sh;
`
	WFSArgsWithoutPrefetch = `set -e;
mkdir -p /srv/data/config/;
rclone config create --non-interactive --obscure blobs azureblob endpoint $BLOBS_ENDPOINT account $BLOBS_ACCOUNT key $BLOBS_KEY use_emulator true;
`

	WMSArgsForGeoPackageLayers = `set -e;
mkdir -p /srv/data/config/;
rclone config create --non-interactive --obscure blobs azureblob endpoint $BLOBS_ENDPOINT account $BLOBS_ACCOUNT key $BLOBS_KEY use_emulator true;
bash /srv/scripts/gpkg_download.sh;
rclone copyto blobs:/resources-bucket/key/gpkg-symbol.png /srv/data/images/gpkg-symbol.png || exit 1;
rclone copyto blobs:/resources-bucket/key/symbol.svg /srv/data/images/symbol.svg || exit 1;
rclone copyto blobs:/resources-bucket/key/font-1.ttf /srv/data/config/fonts/font-1.ttf || exit 1;
echo font-1 font-1.ttf >> /srv/data/config/fonts/fonts.list;
rclone copyto blobs:/resources-bucket/key/font-2.ttf /srv/data/config/fonts/font-2.ttf || exit 1;
echo font-2 font-2.ttf >> /srv/data/config/fonts/fonts.list;
echo 'generated fonts.list:';
cat /srv/data/config/fonts/fonts.list;
mkdir -p /var/www/legend/wms-gpkg-layer-1-name;
rclone copyto blobs:/resources-bucket/key/gpkg-layer-1-legend.png /var/www/legend/wms-gpkg-layer-1-name/wms-gpkg-style-1-name.png || exit 1;
Copied legend gpkg-layer-1-legend.png to /var/www/legend/wms-gpkg-layer-1-name/wms-gpkg-style-1-name.png;
mkdir -p /var/www/legend/wms-gpkg-layer-2-name;
rclone copyto blobs:/resources-bucket/key/gpkg-layer-2-legend.png /var/www/legend/wms-gpkg-layer-2-name/wms-gpkg-style-2-name.png || exit 1;
Copied legend gpkg-layer-2-legend.png to /var/www/legend/wms-gpkg-layer-2-name/wms-gpkg-style-2-name.png;
chown -R 999:999 /var/www/legend
`

	WMSArgsForTIFLayers = `set -e;
mkdir -p /srv/data/config/;
rclone config create --non-interactive --obscure blobs azureblob endpoint $BLOBS_ENDPOINT account $BLOBS_ACCOUNT key $BLOBS_KEY use_emulator true;
bash /srv/scripts/gpkg_download.sh;
rclone copyto blobs:/tifs-bucket/key/tif-layer-1-data.tif /srv/data/tif/tif-layer-1-data.tif || exit 1;
rclone copyto blobs:/tifs-bucket/key/tif-layer-2-data.tif /srv/data/tif/tif-layer-2-data.tif || exit 1;
rclone copyto blobs:/resources-bucket/key/tif-symbol.png /srv/data/images/tif-symbol.png || exit 1;
rclone copyto blobs:/resources-bucket/key/symbol.svg /srv/data/images/symbol.svg || exit 1;
rclone copyto blobs:/resources-bucket/key/font-1.ttf /srv/data/config/fonts/font-1.ttf || exit 1;
echo font-1 font-1.ttf >> /srv/data/config/fonts/fonts.list;
rclone copyto blobs:/resources-bucket/key/font-2.ttf /srv/data/config/fonts/font-2.ttf || exit 1;
echo font-2 font-2.ttf >> /srv/data/config/fonts/fonts.list;
echo 'generated fonts.list:';
cat /srv/data/config/fonts/fonts.list;
mkdir -p /var/www/legend/wms-tif-layer-1-name;
rclone copyto blobs:/resources-bucket/key/tif-layer-1-legend.png /var/www/legend/wms-tif-layer-1-name/wms-tif-style-1-name.png || exit 1;
Copied legend tif-layer-1-legend.png to /var/www/legend/wms-tif-layer-1-name/wms-tif-style-1-name.png;
mkdir -p /var/www/legend/wms-tif-layer-2-name;
rclone copyto blobs:/resources-bucket/key/tif-layer-2-legend.png /var/www/legend/wms-tif-layer-2-name/wms-tif-style-2-name.png || exit 1;
Copied legend tif-layer-2-legend.png to /var/www/legend/wms-tif-layer-2-name/wms-tif-style-2-name.png;
chown -R 999:999 /var/www/legend
`
)

func TestGetArgsForWFS(t *testing.T) {
	type args struct {
		WFS *pdoknlv3.WFS
	}
	tests := []struct {
		name     string
		args     args
		wantArgs string
		wantErr  bool
	}{
		{
			name: "GetArgs for WFS with prefetchData",
			args: args{
				WFS: &pdoknlv3.WFS{
					Spec: pdoknlv3.WFSSpec{
						Service: pdoknlv3.WFSService{BaseService: pdoknlv3.BaseService{
							Title: "wfs-prefetch-service-title",
						}},
						Options: &pdoknlv3.BaseOptions{
							PrefetchData: true,
						},
					},
				},
			},
			wantArgs: WFSArgsWithPrefetch,
			wantErr:  false,
		},
		{
			name: "GetArgs for WFS without prefetchData",
			args: args{
				WFS: &pdoknlv3.WFS{
					Spec: pdoknlv3.WFSSpec{
						Service: pdoknlv3.WFSService{BaseService: pdoknlv3.BaseService{
							Title: "wfs-noprefetch-service-title",
						}},
						Options: &pdoknlv3.BaseOptions{
							PrefetchData: false,
						},
					},
				},
			},
			wantArgs: WFSArgsWithoutPrefetch,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := GetArgs(tt.args.WFS)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if args == "" {
				t.Errorf("The returned arguments are empty.")
			}
			if args != tt.wantArgs {
				t.Errorf("GetArgs() = %v, want %v", args, tt.wantArgs)
				return
			}
		})
	}
}

func TestGetArgsForWMS(t *testing.T) {
	type args struct {
		WMS pdoknlv3.WMS
	}
	tests := []struct {
		name     string
		args     args
		wantArgs string
		wantErr  bool
	}{
		{
			name: "GetArgs for WMS GeoPackage layer",
			args: args{
				WMS: pdoknlv3.WMS{
					Spec: pdoknlv3.WMSSpec{
						Service: pdoknlv3.WMSService{BaseService: pdoknlv3.BaseService{
							Title: "wms-gpkg-service-title"},
							Layer: pdoknlv3.Layer{
								Name:  smoothoperatorutils.Pointer("wms-gpkg-layer-name"),
								Title: smoothoperatorutils.Pointer("wms-gpkg-layer-title"),
								Styles: []pdoknlv3.Style{
									{
										Legend: &pdoknlv3.Legend{
											BlobKey: "key/gpkg-layer-legend.png",
										},
									},
								},
								Layers: []pdoknlv3.Layer{
									{
										Name:  smoothoperatorutils.Pointer("wms-gpkg-layer-1-name"),
										Title: smoothoperatorutils.Pointer("wms-gpkg-layer-1-title"),
										Styles: []pdoknlv3.Style{
											{
												Name:  "wms-gpkg-style-1-name",
												Title: smoothoperatorutils.Pointer("wms-gpkg-style-1-title"),
												Legend: &pdoknlv3.Legend{
													Width:   50,
													Height:  50,
													Format:  "png",
													BlobKey: "resources-bucket/key/gpkg-layer-1-legend.png",
												},
											},
										},
										Data: &pdoknlv3.Data{BaseData: pdoknlv3.BaseData{
											Gpkg: &pdoknlv3.Gpkg{
												BlobKey: "geopackages-bucket/key/gpkg-layer-1-data.gpkg",
											}},
										},
									},
									{
										Name:  smoothoperatorutils.Pointer("wms-gpkg-layer-2-name"),
										Title: smoothoperatorutils.Pointer("wms-gpkg-layer-2-title"),
										Styles: []pdoknlv3.Style{
											{
												Name:  "wms-gpkg-style-2-name",
												Title: smoothoperatorutils.Pointer("wms-gpkg-style-2-title"),
												Legend: &pdoknlv3.Legend{
													BlobKey: "resources-bucket/key/gpkg-layer-2-legend.png",
												},
											},
										},
										Data: &pdoknlv3.Data{BaseData: pdoknlv3.BaseData{
											Gpkg: &pdoknlv3.Gpkg{
												BlobKey: "geopackages-bucket/key/gpkg-layer-2-data.gpkg",
											}},
										},
									},
								},
							},
							StylingAssets: &pdoknlv3.StylingAssets{
								BlobKeys: []string{
									"resources-bucket/key/gpkg-symbol.png",
									"resources-bucket/key/symbol.svg",
									"resources-bucket/key/font-1.ttf",
									"resources-bucket/key/font-2.ttf",
								},
							},
						},
						Options: &pdoknlv3.Options{BaseOptions: pdoknlv3.BaseOptions{
							PrefetchData: true,
						}},
					},
				},
			},
			wantArgs: WMSArgsForGeoPackageLayers,
			wantErr:  false,
		},
		{
			name: "GetArgs for WMS TIF layer",
			args: args{
				WMS: pdoknlv3.WMS{
					Spec: pdoknlv3.WMSSpec{
						Service: pdoknlv3.WMSService{BaseService: pdoknlv3.BaseService{
							Title: "wms-tif-service-title"},
							Layer: pdoknlv3.Layer{
								Name:  smoothoperatorutils.Pointer("wms-tif-layer-name"),
								Title: smoothoperatorutils.Pointer("wms-tif-layer-title"),
								Layers: []pdoknlv3.Layer{
									{
										Name:  smoothoperatorutils.Pointer("wms-tif-layer-1-name"),
										Title: smoothoperatorutils.Pointer("wms-tif-layer-1-title"),
										Styles: []pdoknlv3.Style{
											{
												Name:  "wms-tif-style-1-name",
												Title: smoothoperatorutils.Pointer("wms-tif-style-1-title"),
												Legend: &pdoknlv3.Legend{
													BlobKey: "resources-bucket/key/tif-layer-1-legend.png",
												},
											},
										},
										Data: &pdoknlv3.Data{
											TIF: &pdoknlv3.TIF{
												BlobKey: "tifs-bucket/key/tif-layer-1-data.tif",
											},
										},
									},
									{
										Name:  smoothoperatorutils.Pointer("wms-tif-layer-2-name"),
										Title: smoothoperatorutils.Pointer("wms-tif-layer-2-title"),
										Styles: []pdoknlv3.Style{
											{
												Name:  "wms-tif-style-2-name",
												Title: smoothoperatorutils.Pointer("wms-tif-style-2-title"),
												Legend: &pdoknlv3.Legend{
													BlobKey: "resources-bucket/key/tif-layer-2-legend.png",
												},
											},
										},
										Data: &pdoknlv3.Data{
											TIF: &pdoknlv3.TIF{
												BlobKey: "tifs-bucket/key/tif-layer-2-data.tif",
											},
										},
									},
								},
							},
							StylingAssets: &pdoknlv3.StylingAssets{
								BlobKeys: []string{
									"resources-bucket/key/tif-symbol.png",
									"resources-bucket/key/symbol.svg",
									"resources-bucket/key/font-1.ttf",
									"resources-bucket/key/font-2.ttf",
								},
							},
						},
						Options: &pdoknlv3.Options{BaseOptions: pdoknlv3.BaseOptions{
							PrefetchData: true,
						}},
					},
				},
			},
			wantArgs: WMSArgsForTIFLayers,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := GetArgs(&tt.args.WMS)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if args != tt.wantArgs {
				diff := cmp.Diff(tt.wantArgs, args)
				t.Errorf("GetArgs() -want, +got %s", diff)
				return
			}
		})
	}
}

func TestGetScript(t *testing.T) {
	tests := []struct {
		name          string
		wantHeader    string
		wantFunctions []string
		wantErr       bool
	}{
		{
			name:          "Test for expected header and functions in the returned bash script",
			wantHeader:    "#!/usr/bin/env bash",
			wantFunctions: []string{"download_gpkg", "download", "download_all", "rm_file_and_exit"},
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := GetScript()
			if !strings.HasPrefix(script, tt.wantHeader) {
				t.Errorf("The returned script doesn't contain the expected header `%v`, got = %v", tt.wantHeader, script)
			}

			for _, function := range tt.wantFunctions {
				funcString := "function " + function + "()"
				if !strings.Contains(script, funcString) {
					t.Errorf("The returned script doesn't contain the expected function `%v`, got = %v", funcString, script)
				}
			}
		})
	}
}
