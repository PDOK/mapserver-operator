package controller

import (
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/constants"
	"github.com/pdok/mapserver-operator/internal/controller/mapperutils"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func mutateHorizontalPodAutoscaler[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, autoscaler *autoscalingv2.HorizontalPodAutoscaler) error {
	reconcilerClient := getReconcilerClient(r)
	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, autoscaler, labels); err != nil {
		return err
	}

	autoscaler.Spec.MaxReplicas = 30
	autoscaler.Spec.MinReplicas = smoothoperatorutils.Pointer(int32(2))
	autoscaler.Spec.ScaleTargetRef = autoscalingv2.CrossVersionObjectReference{
		APIVersion: appsv1.SchemeGroupVersion.String(),
		Kind:       "Deployment",
		Name:       getSuffixedName(obj, constants.MapserverName),
	}

	var averageCPU int32 = 90
	if cpu := mapperutils.GetContainerResourceRequest(obj, constants.MapserverName, corev1.ResourceCPU); cpu != nil {
		averageCPU = 80
	}
	autoscaler.Spec.Metrics = []autoscalingv2.MetricSpec{{
		Type: autoscalingv2.ResourceMetricSourceType,
		Resource: &autoscalingv2.ResourceMetricSource{
			Name: corev1.ResourceCPU,
			Target: autoscalingv2.MetricTarget{
				Type:               autoscalingv2.UtilizationMetricType,
				AverageUtilization: &averageCPU,
			},
		},
	}}

	var behaviourStabilizationWindowSeconds int32
	if obj.Type() == pdoknlv3.ServiceTypeWFS {
		behaviourStabilizationWindowSeconds = 300
	}

	autoscaler.Spec.Behavior = &autoscalingv2.HorizontalPodAutoscalerBehavior{
		ScaleUp: &autoscalingv2.HPAScalingRules{
			StabilizationWindowSeconds: &behaviourStabilizationWindowSeconds,
			Policies: []autoscalingv2.HPAScalingPolicy{{
				Type:          autoscalingv2.PodsScalingPolicy,
				Value:         20,
				PeriodSeconds: 60,
			}},
			SelectPolicy: smoothoperatorutils.Pointer(autoscalingv2.MaxChangePolicySelect),
		},
		ScaleDown: &autoscalingv2.HPAScalingRules{
			StabilizationWindowSeconds: smoothoperatorutils.Pointer(int32(3600)),
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
			SelectPolicy: smoothoperatorutils.Pointer(autoscalingv2.MaxChangePolicySelect),
		},
	}
	if obj.HorizontalPodAutoscalerPatch() != nil {
		patchedSpec, err := smoothoperatorutils.StrategicMergePatch(&autoscaler.Spec, obj.HorizontalPodAutoscalerPatch())
		if err != nil {
			return err
		}
		autoscaler.Spec = *patchedSpec
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
