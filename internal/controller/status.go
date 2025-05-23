package controller

import (
	"context"
	"time"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func logAndUpdateStatusError[R Reconciler](ctx context.Context, r R, obj client.Object, err error) {
	var generation int64

	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		generation = any(obj).(*pdoknlv3.WFS).Generation
	case *pdoknlv3.WMS:
		generation = any(obj).(*pdoknlv3.WMS).Generation
	}

	updateStatus(ctx, r, obj, []metav1.Condition{{
		Type:               reconciledConditionType,
		Status:             metav1.ConditionFalse,
		Reason:             reconciledConditionReasonError,
		Message:            err.Error(),
		ObservedGeneration: generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}}, nil)
}

func logAndUpdateStatusFinished[R Reconciler](ctx context.Context, r R, obj client.Object, operationResults map[string]controllerutil.OperationResult) {
	lgr := log.FromContext(ctx)
	lgr.Info("operation results", "results", operationResults)

	var generation int64

	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		generation = any(obj).(*pdoknlv3.WFS).Generation
	case *pdoknlv3.WMS:
		generation = any(obj).(*pdoknlv3.WMS).Generation
	}

	updateStatus(ctx, r, obj, []metav1.Condition{{
		Type:               reconciledConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             reconciledConditionReasonSuccess,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}}, operationResults)
}

func updateStatus[R Reconciler](ctx context.Context, r R, obj client.Object, conditions []metav1.Condition, operationResults map[string]controllerutil.OperationResult) {
	lgr := log.FromContext(ctx)
	if err := getReconcilerClient(r).Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		log.FromContext(ctx).Error(err, "unable to update status")
		return
	}

	var status *smoothoperatormodel.OperatorStatus
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		status = &any(obj).(*pdoknlv3.WFS).Status
	case *pdoknlv3.WMS:
		status = &any(obj).(*pdoknlv3.WMS).Status
	}

	changed := false
	for _, condition := range conditions {
		if meta.SetStatusCondition(&status.Conditions, condition) {
			changed = true
		}
	}
	if !equality.Semantic.DeepEqual(status.OperationResults, operationResults) {
		status.OperationResults = operationResults
		changed = true
	}
	if !changed {
		return
	}
	if err := r.Status().Update(ctx, obj); err != nil {
		lgr.Error(err, "unable to update status")
	}
}
