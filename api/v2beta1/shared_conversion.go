package v2beta1

import (
	"fmt"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	shared_model "github.com/pdok/smooth-operator/model"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"
)

func Pointer[T interface{}](val T) *T {
	return &val
}

func PointerVal[T interface{}](val *T, def T) T {
	if val == nil {
		return def
	} else {
		return *val
	}
}

func ConverseOptionsV2ToV3(src WMSWFSOptions) *pdoknlv3.Options {
	return &pdoknlv3.Options{
		AutomaticCasing:             src.AutomaticCasing,
		IncludeIngress:              src.IncludeIngress,
		PrefetchData:                src.PrefetchData,
		ValidateRequests:            src.ValidateRequests,
		RewriteGroupToDataLayers:    src.RewriteGroupToDataLayers,
		DisableWebserviceProxy:      src.DisableWebserviceProxy,
		ValidateChildStyleNameEqual: src.ValidateChildStyleNameEqual,
	}
}

func ConverseOptionsV3ToV2(src *pdoknlv3.Options) WMSWFSOptions {
	return WMSWFSOptions{
		AutomaticCasing:             src.AutomaticCasing,
		PrefetchData:                src.PrefetchData,
		IncludeIngress:              src.IncludeIngress,
		ValidateRequests:            src.ValidateRequests,
		RewriteGroupToDataLayers:    src.RewriteGroupToDataLayers,
		DisableWebserviceProxy:      src.DisableWebserviceProxy,
		ValidateChildStyleNameEqual: src.ValidateChildStyleNameEqual,
	}
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

func ConverseV2DataToV3(v2 Data) pdoknlv3.Data {
	v3 := pdoknlv3.Data{}

	if v2.GPKG != nil {
		v3.Gpkg = &pdoknlv3.Gpkg{
			BlobKey:      v2.GPKG.BlobKey,
			TableName:    v2.GPKG.Table,
			GeometryType: v2.GPKG.GeometryType,
			Columns: ConverseColumnAndAliasesV2ToColumnsWithAliasV3(
				v2.GPKG.Columns,
				v2.GPKG.Aliases,
			),
		}
	}

	if v2.Postgis != nil {
		v3.Postgis = &pdoknlv3.Postgis{
			TableName:    v2.Postgis.Table,
			GeometryType: v2.Postgis.GeometryType,
			Columns: ConverseColumnAndAliasesV2ToColumnsWithAliasV3(
				v2.Postgis.Columns,
				v2.Postgis.Aliases,
			),
		}
	}

	if v2.Tif != nil {
		v3.TIF = &pdoknlv3.TIF{
			BlobKey:                     v2.Tif.BlobKey,
			Resample:                    v2.Tif.Resample,
			Offsite:                     v2.Tif.Offsite,
			GetFeatureInfoIncludesClass: v2.Tif.GetFeatureInfoIncludesClass,
		}
	}

	return v3
}

func ConverseV3DataToV2(v3 pdoknlv3.Data) Data {
	v2 := Data{}

	if v3.Gpkg != nil {
		columns, aliases := ConverseColumnsWithAliasV3ToColumnsAndAliasesV2(v3.Gpkg.Columns)
		v2.GPKG = &GPKG{
			BlobKey:      v3.Gpkg.BlobKey,
			Table:        v3.Gpkg.TableName,
			GeometryType: v3.Gpkg.GeometryType,
			Columns:      columns,
			Aliases:      aliases,
		}
	}

	if v3.Postgis != nil {
		columns, aliases := ConverseColumnsWithAliasV3ToColumnsAndAliasesV2(v3.Postgis.Columns)
		v2.Postgis = &Postgis{
			Table:        v3.Postgis.TableName,
			GeometryType: v3.Postgis.GeometryType,
			Columns:      columns,
			Aliases:      aliases,
		}
	}

	if v3.TIF != nil {
		v2.Tif = &Tif{
			BlobKey:                     v3.TIF.BlobKey,
			Offsite:                     v3.TIF.Offsite,
			Resample:                    v3.TIF.Resample,
			GetFeatureInfoIncludesClass: v3.TIF.GetFeatureInfoIncludesClass,
		}
	}

	return v2
}

func NewV2KubernetesObject(lifecycle *shared_model.Lifecycle, podSpecPatch *corev1.PodSpec, scalingSpec *autoscalingv2.HorizontalPodAutoscalerSpec) Kubernetes {
	kub := Kubernetes{}

	if lifecycle != nil && lifecycle.TTLInDays != nil {
		kub.Lifecycle = &Lifecycle{
			TTLInDays: Pointer(int(*lifecycle.TTLInDays)),
		}
	}

	// TODO - healthcheck
	if podSpecPatch != nil {
		kub.Resources = &podSpecPatch.Containers[0].Resources
	}

	if scalingSpec != nil {
		kub.Autoscaling = &Autoscaling{
			MaxReplicas: Pointer(int(scalingSpec.MaxReplicas)),
		}

		if scalingSpec.MinReplicas != nil {
			kub.Autoscaling.MinReplicas = Pointer(int(*scalingSpec.MinReplicas))
		}

		if scalingSpec.Metrics != nil {
			kub.Autoscaling.AverageCPUUtilization = Pointer(
				int(*scalingSpec.Metrics[0].Resource.TargetAverageUtilization),
			)
		}
	}

	return kub
}

func LabelsToV2General(labels map[string]string) General {
	general := General{
		Dataset:      labels["dataset"],
		DatasetOwner: labels["dataset-owner"],
		DataVersion:  nil,
	}

	if serviceVersion, ok := labels["service-version"]; ok {
		general.ServiceVersion = &serviceVersion
	}

	if theme, ok := labels["theme"]; ok {
		general.Theme = &theme
	}

	return general
}

func CreateBaseURL(host string, kind string, general General) string {
	URI := fmt.Sprintf("%s/%s", general.DatasetOwner, general.Dataset)
	if general.Theme != nil {
		URI += "/" + *general.Theme
	}
	URI += "/" + kind

	if general.ServiceVersion != nil {
		URI += "/" + *general.ServiceVersion
	}

	return fmt.Sprintf("%s/%s", host, URI)
}
