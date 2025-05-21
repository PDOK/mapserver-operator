package controller

import (
	"errors"
	"strings"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getBareIngressRoute[O pdoknlv3.WMSWFS](obj O) *traefikiov1alpha1.IngressRoute {
	return &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSuffixedName(obj, MapserverName),
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateIngressRoute[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, ingressRoute *traefikiov1alpha1.IngressRoute) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, ingressRoute, labels); err != nil {
		return err
	}

	var uptimeURL string
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		uptimeURL = any(obj).(*pdoknlv3.WFS).Spec.Service.URL // TODO add healthcheck query
	case *pdoknlv3.WMS:
		uptimeURL = any(obj).(*pdoknlv3.WMS).Spec.Service.URL // TODO add healthcheck query
	}

	uptimeName, err := makeUptimeName(obj)
	if err != nil {
		return err
	}
	annotations := smoothoperatorutils.CloneOrEmptyMap(obj.GetAnnotations())
	annotations["uptime.pdok.nl/id"] = obj.ID()
	annotations["uptime.pdok.nl/name"] = uptimeName
	annotations["uptime.pdok.nl/url"] = uptimeURL
	annotations["uptime.pdok.nl/tags"] = strings.Join(makeUptimeTags(obj), ",")
	ingressRoute.SetAnnotations(annotations)

	mapserverService := traefikiov1alpha1.Service{
		LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
			Name: getBareService(obj).GetName(),
			Kind: "Service",
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: mapserverPortNr,
			},
		},
	}

	middlewareRef := traefikiov1alpha1.MiddlewareRef{
		Name: getBareCorsHeadersMiddleware(obj).GetName(),
	}

	if obj.Type() == pdoknlv3.ServiceTypeWMS {
		wms, _ := any(obj).(*pdoknlv3.WMS)
		ingressRoute.Spec.Routes = []traefikiov1alpha1.Route{{
			Kind:        "Rule",
			Match:       getLegendMatchRule(wms),
			Services:    []traefikiov1alpha1.Service{mapserverService},
			Middlewares: []traefikiov1alpha1.MiddlewareRef{middlewareRef},
		}}

		if obj.Options().UseWebserviceProxy() {
			webServiceProxyService := traefikiov1alpha1.Service{
				LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
					Name: getBareService(obj).GetName(),
					Kind: "Service",
					Port: intstr.IntOrString{
						Type: intstr.Int,

						IntVal: int32(mapserverWebserviceProxyPortNr),
					},
				},
			}

			ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, traefikiov1alpha1.Route{
				Kind:        "Rule",
				Match:       getMatchRule(obj),
				Services:    []traefikiov1alpha1.Service{webServiceProxyService},
				Middlewares: []traefikiov1alpha1.MiddlewareRef{middlewareRef},
			})
		} else {
			ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, traefikiov1alpha1.Route{
				Kind:        "Rule",
				Match:       getMatchRule(obj),
				Services:    []traefikiov1alpha1.Service{mapserverService},
				Middlewares: []traefikiov1alpha1.MiddlewareRef{middlewareRef},
			})
		}
	} else { // WFS
		ingressRoute.Spec.Routes = []traefikiov1alpha1.Route{{
			Kind:        "Rule",
			Match:       getMatchRule(obj),
			Services:    []traefikiov1alpha1.Service{mapserverService},
			Middlewares: []traefikiov1alpha1.MiddlewareRef{middlewareRef},
		}}
	}

	// Add finalizers
	ingressRoute.Finalizers = []string{"uptime.pdok.nl/finalizer"}

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, ingressRoute, ingressRoute); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, ingressRoute, getReconcilerScheme(r))
}

func makeUptimeTags[O pdoknlv3.WMSWFS](obj O) []string {
	tags := []string{"public-stats", strings.ToLower(string(obj.Type()))}

	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		wfs, _ := any(obj).(*pdoknlv3.WFS)
		if wfs.Spec.Service.Inspire != nil {
			tags = append(tags, "inspire")
		}
	case *pdoknlv3.WMS:
		wms, _ := any(obj).(*pdoknlv3.WMS)
		if wms.Spec.Service.Inspire != nil {
			tags = append(tags, "inspire")
		}
	}

	return tags
}

func makeUptimeName[O pdoknlv3.WMSWFS](obj O) (string, error) {
	var parts []string

	inspire := false
	switch any(obj).(type) {
	case *pdoknlv3.WFS:
		inspire = any(obj).(*pdoknlv3.WFS).Spec.Service.Inspire != nil
	case *pdoknlv3.WMS:
		inspire = any(obj).(*pdoknlv3.WMS).Spec.Service.Inspire != nil
	}

	ownerID, ok := obj.GetLabels()["dataset-owner"]
	if !ok {
		ownerID, ok = obj.GetLabels()["pdok.nl/owner-id"]
		if !ok {
			return "", errors.New("dataset-owner and pdok.nl/owner-id labels are not found in object")
		}
	}
	parts = append(parts, strings.ToUpper(strings.ReplaceAll(ownerID, "-", "")))

	datasetID, ok := obj.GetLabels()["dataset"]
	if !ok {
		// V3 label
		datasetID, ok = obj.GetLabels()["pdok.nl/dataset-id"]
		if !ok {
			return "", errors.New("dataset label not found in object")
		}
	}
	parts = append(parts, strings.ReplaceAll(datasetID, "-", ""))

	theme, ok := obj.GetLabels()["theme"]
	if !ok {
		// V3 label
		theme, ok = obj.GetLabels()["pdok.nl/tag"]
	}

	if ok {
		parts = append(parts, strings.ReplaceAll(theme, "-", ""))
	}

	version, ok := obj.GetLabels()["service-version"]
	if !ok {
		version, ok = obj.GetLabels()["pdok.nl/service-version"]
		if !ok {
			return "", errors.New("service-version label not found in object")
		}
	}
	parts = append(parts, version)

	if inspire {
		parts = append(parts, "INSPIRE")
	}

	parts = append(parts, string(obj.Type()))

	return strings.Join(parts, " "), nil
}

func getMatchRule[O pdoknlv3.WMSWFS](obj O) string {
	host := pdoknlv3.GetHost(false)
	if strings.Contains(host, "localhost") {
		return "Host(`localhost`) && Path(`/" + pdoknlv3.GetBaseURLPath(obj) + "`)"
	}

	return "(Host(`localhost`) || Host(`" + host + "`)) && Path(`/" + pdoknlv3.GetBaseURLPath(obj) + "`)"
}

func getLegendMatchRule(wms *pdoknlv3.WMS) string {
	host := pdoknlv3.GetHost(false)
	if strings.Contains(host, "localhost") {
		return "Host(`localhost`) && PathPrefix(`/" + pdoknlv3.GetBaseURLPath(wms) + "/legend`)"
	}

	return "(Host(`localhost`) || Host(`" + host + "`)) && PathPrefix(`/" + pdoknlv3.GetBaseURLPath(wms) + "/legend`)"
}
