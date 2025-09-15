package controller

import (
	"fmt"
	"strings"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/blobdownload"
	"github.com/pdok/mapserver-operator/internal/controller/capabilitiesgenerator"
	"github.com/pdok/mapserver-operator/internal/controller/mapfilegenerator"
	"github.com/pdok/mapserver-operator/internal/controller/static"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	downloadScriptName         = "gpkg_download.sh"
	mapfileGeneratorInput      = "input.json"
	capabilitiesGeneratorInput = "input.yaml"
)

func mutateConfigMapCapabilitiesGenerator[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		input, err := capabilitiesgenerator.GetInput(obj, ownerInfo)
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{capabilitiesGeneratorInput: input}
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(obj, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func mutateConfigMapMapfileGenerator[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		mapfileGeneratorConfig, err := mapfilegenerator.GetConfig(obj, ownerInfo)
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{mapfileGeneratorInput: mapfileGeneratorConfig}
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(obj, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func mutateConfigMapBlobDownload[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		downloadScript := blobdownload.GetScript()
		configMap.Data = map[string]string{downloadScriptName: downloadScript}
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(obj, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func getBareConfigMap[O pdoknlv3.WMSWFS](obj O, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSuffixedName(obj, name),
			Namespace: obj.GetNamespace(),
		},
	}
}

func mutateConfigMap[R Reconciler, O pdoknlv3.WMSWFS](r R, obj O, configMap *corev1.ConfigMap) error {
	reconcilerClient := getReconcilerClient(r)

	labels := addCommonLabels(obj, smoothoperatorutils.CloneOrEmptyMap(obj.GetLabels()))
	if err := smoothoperatorutils.SetImmutableLabels(reconcilerClient, configMap, labels); err != nil {
		return err
	}

	configMap.Immutable = smoothoperatorutils.Pointer(true)
	configMap.Data = map[string]string{}

	updateConfigMapWithStaticFiles(configMap, obj)

	if err := smoothoperatorutils.EnsureSetGVK(reconcilerClient, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(obj, configMap, getReconcilerScheme(r)); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func updateConfigMapWithStaticFiles[O pdoknlv3.WMSWFS](configMap *corev1.ConfigMap, obj O) {
	staticFileName, contents := static.GetStaticFiles()
	for _, name := range staticFileName {
		content := contents[name]
		if name == "include.conf" {
			ingressRouteUrls := obj.IngressRouteURLs(true)
			rewriteRules := make([]string, 0)
			for _, ingressRouteUrl := range ingressRouteUrls {
				rewriteRules = append(rewriteRules, fmt.Sprintf("  \"%s/legend(.*)\" => \"/legend$1\"", ingressRouteUrl.URL.Path))
				rewriteRules = append(rewriteRules, fmt.Sprintf("  \"%s/(.*)\" => \"/mapserver$1\"", ingressRouteUrl.URL.Path))
			}

			content = []byte(strings.ReplaceAll(string(content), "{{ rewrite_rules }}", strings.Join(rewriteRules, ",\n")))
		}
		configMap.Data[name] = string(content)
	}
}
