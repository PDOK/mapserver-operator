package v2beta1

import (
	"net/url"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/constants"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
)

func ConvertOptionsV2ToV3(src *WMSWFSOptions) *pdoknlv3.Options {
	defaults := pdoknlv3.GetDefaultOptions()

	if src == nil {
		return defaults
	}

	return &pdoknlv3.Options{
		AutomaticCasing:             src.AutomaticCasing,
		IncludeIngress:              src.IncludeIngress,
		PrefetchData:                smoothoperatorutils.PointerVal(src.PrefetchData, defaults.PrefetchData),
		ValidateRequests:            smoothoperatorutils.PointerVal(src.ValidateRequests, defaults.ValidateRequests),
		RewriteGroupToDataLayers:    smoothoperatorutils.PointerVal(src.RewriteGroupToDataLayers, defaults.RewriteGroupToDataLayers),
		DisableWebserviceProxy:      smoothoperatorutils.PointerVal(src.DisableWebserviceProxy, defaults.DisableWebserviceProxy),
		ValidateChildStyleNameEqual: smoothoperatorutils.PointerVal(src.ValidateChildStyleNameEqual, defaults.ValidateChildStyleNameEqual),
	}
}

func ConvertOptionsV3ToV2(src *pdoknlv3.Options) *WMSWFSOptions {
	if src == nil {
		src = pdoknlv3.GetDefaultOptions()
	}

	return &WMSWFSOptions{
		AutomaticCasing:             src.AutomaticCasing,
		IncludeIngress:              src.IncludeIngress,
		PrefetchData:                &src.PrefetchData,
		ValidateRequests:            &src.ValidateRequests,
		RewriteGroupToDataLayers:    &src.RewriteGroupToDataLayers,
		DisableWebserviceProxy:      &src.DisableWebserviceProxy,
		ValidateChildStyleNameEqual: &src.ValidateChildStyleNameEqual,
	}
}

//nolint:gosec
func ConvertAutoscaling(src Autoscaling) *pdoknlv3.HorizontalPodAutoscalerPatch {
	var minReplicas *int32
	if src.MinReplicas != nil {
		//nolint:gosec
		minReplicas = smoothoperatorutils.Pointer(int32(*src.MinReplicas))
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
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: smoothoperatorutils.Pointer(int32(*src.AverageCPUUtilization)),
				},
			},
		})
	}

	return &pdoknlv3.HorizontalPodAutoscalerPatch{
		MinReplicas: minReplicas,
		MaxReplicas: &maxReplicas,
		Metrics:     metrics,
	}
}

func ConvertResources(src corev1.ResourceRequirements) corev1.PodSpec {
	targetResources := src

	if src.Requests != nil {
		targetResources.Requests[corev1.ResourceEphemeralStorage] = src.Requests["ephemeralStorage"]
		delete(targetResources.Requests, "ephemeralStorage")
	}
	if src.Limits != nil {
		targetResources.Limits[corev1.ResourceEphemeralStorage] = src.Limits["ephemeralStorage"]
		delete(targetResources.Limits, "ephemeralStorage")
	}

	return corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:      constants.MapserverName,
				Resources: targetResources,
			},
		},
	}
}

func ConvertColumnAndAliasesV2ToColumnsWithAliasV3(columns []string, aliases map[string]string) []pdoknlv3.Column {
	v3Columns := make([]pdoknlv3.Column, 0)
	for _, column := range columns {
		col := pdoknlv3.Column{
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

func ConvertColumnsWithAliasV3ToColumnsAndAliasesV2(columns []pdoknlv3.Column) ([]string, map[string]string) {
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

func ConvertV2DataToV3(v2 Data) pdoknlv3.Data {
	v3 := pdoknlv3.Data{}

	if v2.GPKG != nil {
		v3.Gpkg = &pdoknlv3.Gpkg{
			BlobKey:      v2.GPKG.BlobKey,
			TableName:    v2.GPKG.Table,
			GeometryType: v2.GPKG.GeometryType,
			Columns: ConvertColumnAndAliasesV2ToColumnsWithAliasV3(
				v2.GPKG.Columns,
				v2.GPKG.Aliases,
			),
		}
	}

	if v2.Postgis != nil {
		v3.Postgis = &pdoknlv3.Postgis{
			TableName:    v2.Postgis.Table,
			GeometryType: v2.Postgis.GeometryType,
			Columns: ConvertColumnAndAliasesV2ToColumnsWithAliasV3(
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
			GetFeatureInfoIncludesClass: smoothoperatorutils.PointerVal(v2.Tif.GetFeatureInfoIncludesClass, false),
		}
	}

	return v3
}

func ConvertV3DataToV2(v3 pdoknlv3.Data) Data {
	v2 := Data{}

	if v3.Gpkg != nil {
		columns, aliases := ConvertColumnsWithAliasV3ToColumnsAndAliasesV2(v3.Gpkg.Columns)
		v2.GPKG = &GPKG{
			BlobKey:      v3.Gpkg.BlobKey,
			Table:        v3.Gpkg.TableName,
			GeometryType: v3.Gpkg.GeometryType,
			Columns:      columns,
			Aliases:      aliases,
		}
	}

	if v3.Postgis != nil {
		columns, aliases := ConvertColumnsWithAliasV3ToColumnsAndAliasesV2(v3.Postgis.Columns)
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
			GetFeatureInfoIncludesClass: &v3.TIF.GetFeatureInfoIncludesClass,
		}
	}

	return v2
}

func NewV2KubernetesObject(lifecycle *smoothoperatormodel.Lifecycle, podSpecPatch corev1.PodSpec, scalingSpec *pdoknlv3.HorizontalPodAutoscalerPatch) Kubernetes {
	kub := Kubernetes{}

	if lifecycle != nil && lifecycle.TTLInDays != nil {
		kub.Lifecycle = &Lifecycle{
			TTLInDays: smoothoperatorutils.Pointer(int(*lifecycle.TTLInDays)),
		}
	}

	kub.Resources = &podSpecPatch.Containers[0].Resources

	if scalingSpec != nil {
		kub.Autoscaling = &Autoscaling{}

		if scalingSpec.MaxReplicas != nil {
			kub.Autoscaling.MaxReplicas = smoothoperatorutils.Pointer(int(*scalingSpec.MaxReplicas))
		}

		if scalingSpec.MinReplicas != nil {
			kub.Autoscaling.MinReplicas = smoothoperatorutils.Pointer(int(*scalingSpec.MinReplicas))
		}

		if scalingSpec.Metrics != nil {
			kub.Autoscaling.AverageCPUUtilization = smoothoperatorutils.Pointer(
				int(*scalingSpec.Metrics[0].Resource.Target.AverageUtilization),
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

func CreateBaseURL(host string, kind string, general General) (*smoothoperatormodel.URL, error) {
	baseURL, err := url.Parse(host + "/")
	if err != nil {
		return nil, err
	}
	baseURL = baseURL.JoinPath(general.DatasetOwner, general.Dataset)
	if general.Theme != nil {
		baseURL = baseURL.JoinPath(*general.Theme)
	}
	baseURL = baseURL.JoinPath(kind)

	if general.ServiceVersion != nil {
		baseURL = baseURL.JoinPath(*general.ServiceVersion)
	}

	return &smoothoperatormodel.URL{URL: baseURL}, nil
}
