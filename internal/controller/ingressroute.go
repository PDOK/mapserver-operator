package controller

import (
	"regexp"
	"strings"

	"github.com/pdok/mapserver-operator/internal/controller/constants"

	"github.com/pdok/mapserver-operator/internal/controller/utils"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

var setUptimeOperatorAnnotations = true

func SetUptimeOperatorAnnotations(set bool) {
	setUptimeOperatorAnnotations = set
}

func getBareIngressRoute[O pdoknlv3.WMSWFS](obj O) *traefikiov1alpha1.IngressRoute {
	return &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSuffixedName(obj, constants.MapserverName),
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

	annotations := smoothoperatorutils.CloneOrEmptyMap(obj.GetAnnotations())
	if setUptimeOperatorAnnotations {
		tags := []string{"public-stats", strings.ToLower(string(obj.Type()))}

		if obj.Inspire() != nil {
			tags = append(tags, "inspire")
		}

		queryString, _, err := obj.ReadinessQueryString()
		if err != nil {
			return err
		}

		annotations["uptime.pdok.nl/id"] = utils.Sha1Hash(obj.TypedName())
		annotations["uptime.pdok.nl/name"] = getUptimeName(obj)
		annotations["uptime.pdok.nl/url"] = obj.URLPath() + "?" + queryString
		annotations["uptime.pdok.nl/tags"] = strings.Join(tags, ",")
	}
	ingressRoute.SetAnnotations(annotations)

	mapserverService := traefikiov1alpha1.Service{
		LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
			Name: getBareService(obj).GetName(),
			Kind: "Service",
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: constants.MapserverPortNr,
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

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, ingressRoute, ingressRoute); err != nil {
		return err
	}
	return ctrl.SetControllerReference(obj, ingressRoute, getReconcilerScheme(r))
}

// getUptimeName transforms the CR name into a uptime.pdok.nl/name value
// owner-dataset-v1-0 -> OWNER dataset v1_0 [INSPIRE] [WMS|WFS]
func getUptimeName[O pdoknlv3.WMSWFS](obj O) string {
	// Extract the version from the CR name, owner-dataset-v1-0 -> owner-dataset + v1-0
	versionMatcher := regexp.MustCompile("^(.*)(?:-(v?[1-9](?:-[0-9])?))?$")
	match := versionMatcher.FindStringSubmatch(obj.GetName())

	nameParts := strings.Split(match[1], "-")
	nameParts[0] = strings.ToUpper(nameParts[0])

	// Add service version if found
	if len(match) > 2 && len(match[2]) > 0 {
		nameParts = append(nameParts, strings.ReplaceAll(match[2], "-", "_"))
	}

	// Add inspire
	if obj.Inspire() != nil {
		nameParts = append(nameParts, "INSPIRE")
	}

	return strings.Join(append(nameParts, string(obj.Type())), " ")
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
