package controller

import (
	"github.com/pdok/mapserver-operator/internal/controller/types"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler interface {
	*WFSReconciler | *WMSReconciler
	client.StatusClient
}

func getReconcilerClient[R Reconciler](r R) client.Client {
	switch any(r).(type) {
	case *WFSReconciler:
		return any(r).(*WFSReconciler).Client
	case *WMSReconciler:
		return any(r).(*WMSReconciler).Client
	}

	return nil
}

func getReconcilerScheme[R Reconciler](r R) *runtime.Scheme {
	switch any(r).(type) {
	case *WFSReconciler:
		return any(r).(*WFSReconciler).Scheme
	case *WMSReconciler:
		return any(r).(*WMSReconciler).Scheme
	}

	return nil
}

func getReconcilerImages[R Reconciler](r R) *types.Images {
	switch any(r).(type) {
	case *WFSReconciler:
		return &any(r).(*WFSReconciler).Images
	case *WMSReconciler:
		return &any(r).(*WMSReconciler).Images
	}

	return nil
}
