package v3

import (
	"reflect"
	"testing"
)

func TestLayer_GetParent(t *testing.T) {
	childLayer2 := Layer{Name: "childlayer-2"}
	childLayer1 := Layer{Name: "childlayer-1", Layers: &[]Layer{childLayer2}}
	topLayer := Layer{Name: "toplayer", Layers: &[]Layer{childLayer1}}

	type args struct {
		candidateLayer *Layer
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
			args:  args{candidateLayer: &topLayer},
			want:  &childLayer1,
		},
		{
			name:  "Test GetParent on layer without parent",
			layer: topLayer,
			args:  args{candidateLayer: &topLayer},
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.layer.GetParent(tt.args.candidateLayer); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetParent() = %v, want %v", got, tt.want)
			}
		})
	}
}
