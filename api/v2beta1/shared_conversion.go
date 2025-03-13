package v2beta1

import (
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"
)

func Pointer[T interface{}](val T) *T {
	return &val
}

func PointerValWithDefault[T interface{}](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}

	return *ptr
}

func ConverseAutoscaling(src Autoscaling) *autoscalingv2.HorizontalPodAutoscalerSpec {
	var minReplicas *int32
	if src.MinReplicas != nil {
		minReplicas = Pointer(int32(*src.MinReplicas))
	}

	var maxReplicas int32
	if src.MaxReplicas != nil {
		maxReplicas = int32(*src.MaxReplicas)
	}

	metrics := make([]autoscalingv2.MetricSpec, 0)
	if src.AverageCPUUtilization != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name:                     corev1.ResourceCPU,
				TargetAverageUtilization: Pointer(int32(*src.AverageCPUUtilization)),
			},
		})
	}

	return &autoscalingv2.HorizontalPodAutoscalerSpec{
		MinReplicas: minReplicas,
		MaxReplicas: maxReplicas,
		Metrics:     metrics,
	}
}

func ConverseResources(src corev1.ResourceRequirements) *corev1.PodSpec {
	return &corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Resources: src,
			},
		},
	}
}

func ConverseColumnAndAliasesV2ToColumnsWithAliasV3(columns []string, aliases map[string]string) []pdoknlv3.Columns {
	v3Columns := make([]pdoknlv3.Columns, 0)
	for _, column := range columns {
		col := pdoknlv3.Columns{
			Name: column,
		}

		// TODO - multiple aliases per column possible?
		if alias, ok := aliases[column]; ok {
			col.Alias = &alias
		}

		v3Columns = append(v3Columns, col)
	}

	return v3Columns
}

func ConverseColumnsWithAliasV3ToColumnsAndAliasesV2(columns []pdoknlv3.Columns) ([]string, map[string]string) {
	v2Columns := make([]string, 0)
	v2Aliases := make(map[string]string)

	for _, col := range columns {
		v2Columns = append(v2Columns, col.Name)

		if col.Alias != nil {
			v2Aliases[col.Name] = *col.Alias
		}
	}

	return v2Columns, v2Aliases
}
