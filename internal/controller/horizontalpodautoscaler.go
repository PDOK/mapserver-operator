package controller

import (
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/constants"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func mutateHorizontalPodAutoscaler[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, autoscaler *autoscalingv2.HorizontalPodAutoscaler) error {
	autoscalerPatch := obj.HorizontalPodAutoscalerPatch()
	var behaviourStabilizationWindowSeconds int32
	if obj.Type() == pdoknlv3.ServiceTypeWFS {
		behaviourStabilizationWindowSeconds = 300
	}

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(getReconcilerClient(r), autoscaler, labels); err != nil {
		return err
	}

	minReplicas := int32(2)
	if autoscalerPatch != nil && autoscalerPatch.MinReplicas != nil {
		minReplicas = *autoscalerPatch.MinReplicas
	}

	maxReplicas := int32(30)
	if autoscalerPatch != nil && autoscalerPatch.MaxReplicas != 0 {
		maxReplicas = autoscalerPatch.MaxReplicas
	}

	var metrics []autoscalingv2.MetricSpec
	if autoscalerPatch != nil {
		metrics = autoscalerPatch.Metrics
	}
	if len(metrics) == 0 {
		var avgU int32 = 90
		if cpu := mapperutils.GetContainerResourceRequest(obj, constants.MapserverName, corev1.ResourceCPU); cpu != nil {
			avgU = 80
		}
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: smoothoperatorutils.Pointer(avgU),
				},
			},
		})
	}

	autoscaler.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       getSuffixedName(obj, constants.MapserverName),
		},
		MinReplicas: &minReplicas,
		MaxReplicas: maxReplicas,
		Metrics:     metrics,
		Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
			ScaleUp: &autoscalingv2.HPAScalingRules{
				StabilizationWindowSeconds: &behaviourStabilizationWindowSeconds,
				SelectPolicy:               smoothoperatorutils.Pointer(autoscalingv2.MaxChangePolicySelect),
				Policies: []autoscalingv2.HPAScalingPolicy{
					{
						Type:          autoscalingv2.PodsScalingPolicy,
						Value:         20,
						PeriodSeconds: 60,
					},
				},
			},
			ScaleDown: &autoscalingv2.HPAScalingRules{
				StabilizationWindowSeconds: smoothoperatorutils.Pointer(int32(3600)),
				SelectPolicy:               smoothoperatorutils.Pointer(autoscalingv2.MaxChangePolicySelect),
				Policies: []autoscalingv2.HPAScalingPolicy{
					{
						Type:          autoscalingv2.PercentScalingPolicy,
						Value:         10,
						PeriodSeconds: 600,
					},
					{
						Type:          autoscalingv2.PodsScalingPolicy,
						Value:         1,
						PeriodSeconds: 600,
					},
				},
			},
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(getReconcilerClient(r), autoscaler, autoscaler); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, autoscaler, getReconcilerScheme(r))
}

func getBareHorizontalPodAutoScaler[O pdoknlv3.WMSWFS](obj O) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSuffixedName(obj, constants.MapserverName),
			Namespace: obj.GetNamespace(),
		},
	}
}
