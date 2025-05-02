package v3

import (
	controller "github.com/pdok/smooth-operator/pkg/util"
	"reflect"
	"testing"
)

func Test_getEqualChildStyleNames(t *testing.T) {
	type args struct {
		layer           *Layer
		equalStyleNames map[string][]string
	}
	tests := []struct {
		name string
		args args
		want map[string][]string
	}{
		{
			name: "Test equal style names",
			args: args{
				layer: &Layer{
					Name: controller.Pointer("toplayer"),
					Styles: []Style{
						{Name: "stylename-1"},
						{Name: "stylename-2"},
					},
					Layers: []Layer{
						{
							Name: controller.Pointer("childlayer-1"),
							Styles: []Style{
								{Name: "stylename-2"},
								{Name: "stylename-3"},
							},
							Layers: []Layer{
								{
									Name: controller.Pointer("childlayer-2"),
									Styles: []Style{
										{Name: "stylename-3"},
										{Name: "stylename-4"},
									},
								},
							},
						},
					},
				},
				equalStyleNames: map[string][]string{},
			},
			want: map[string][]string{
				"childlayer-1": {"stylename-2"},
				"childlayer-2": {"stylename-3"},
			},
		},
		{
			name: "Test no equal style names",
			args: args{
				layer: &Layer{
					Name: controller.Pointer("toplayer"),
					Styles: []Style{
						{Name: "stylename-1"},
						{Name: "stylename-2"},
					},
					Layers: []Layer{
						{
							Name: controller.Pointer("childlayer-1"),
							Styles: []Style{
								{Name: "stylename-3"},
								{Name: "stylename-4"},
							},
							Layers: []Layer{
								{
									Name: controller.Pointer("childlayer-2"),
									Styles: []Style{
										{Name: "stylename-5"},
										{Name: "stylename-6"},
									},
								},
							},
						},
					},
				},
				equalStyleNames: map[string][]string{},
			},
			want: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if findEqualChildStyleNames(tt.args.layer, &tt.args.equalStyleNames); !reflect.DeepEqual(tt.args.equalStyleNames, tt.want) {
				t.Errorf("findEqualChildStyleNames() = %v, want %v", tt.args.equalStyleNames, tt.want)
			}
		})
	}
}
