package v3

import (
	"context"
	"fmt"
	"slices"

	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"
	v1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateCreate[W WMSWFS](c client.Client, obj W, validate func(W, *[]string, *field.ErrorList)) ([]string, error) {
	warnings := []string{}
	allErrs := field.ErrorList{}

	err := sharedValidation.ValidateLabelsOnCreate(obj.GetLabels())
	if err != nil {
		allErrs = append(allErrs, err)
	}

	err = sharedValidation.ValidateIngressRouteURLsContainsBaseURL(obj.IngressRouteURLs(false), obj.URL(), nil)
	if err != nil {
		allErrs = append(allErrs, err)
	}

	validate(obj, &warnings, &allErrs)
	ValidateOwnerInfo(c, obj, &allErrs)

	if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, apierrors.NewInvalid(
		obj.GroupKind(),
		obj.GetName(), allErrs)
}

func ValidateUpdate[W WMSWFS](c client.Client, newW, oldW W, validate func(W, *[]string, *field.ErrorList)) ([]string, error) {
	warnings := []string{}
	allErrs := field.ErrorList{}

	// Make sure no ingressRouteURLs have been removed
	sharedValidation.ValidateIngressRouteURLsNotRemoved(oldW.IngressRouteURLs(false), newW.IngressRouteURLs(true), &allErrs, nil)

	if len(newW.IngressRouteURLs(false)) == 0 {
		// There are no ingressRouteURLs given, spec.service.url is immutable is that case.
		path := field.NewPath("spec").Child("service").Child("url")
		sharedValidation.CheckURLImmutability(
			oldW.URL(),
			newW.URL(),
			&allErrs,
			path,
		)
	} else if oldW.URL().String() != newW.URL().String() {
		// Make sure both the old spec.service.url and the new one are included in the ingressRouteURLs list.
		err := sharedValidation.ValidateIngressRouteURLsContainsBaseURL(newW.IngressRouteURLs(true), oldW.URL(), nil)
		if err != nil {
			allErrs = append(allErrs, err)
		}

		err = sharedValidation.ValidateIngressRouteURLsContainsBaseURL(newW.IngressRouteURLs(true), newW.URL(), nil)
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}

	sharedValidation.ValidateLabelsOnUpdate(oldW.GetLabels(), newW.GetLabels(), &allErrs)

	if (newW.Inspire() == nil && oldW.Inspire() != nil) || (newW.Inspire() != nil && oldW.Inspire() == nil) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("service").Child("inspire"), "cannot change from inspire to not inspire or the other way around"))
	}

	validate(newW, &warnings, &allErrs)
	ValidateOwnerInfo(c, newW, &allErrs)

	if len(allErrs) == 0 {
		return warnings, nil
	}
	return warnings, apierrors.NewInvalid(
		newW.GroupKind(),
		newW.GetName(), allErrs)
}

func ValidateHorizontalPodAutoscalerPatch(patch HorizontalPodAutoscalerPatch, allErrs *field.ErrorList) {
	path := field.NewPath("spec").Child("horizontalPodAutoscaler")
	// TODO: replace hardcoded defaults with dynamic defaults from cli options or ownerInfo
	var minReplicas, maxReplicas int32 = 2, 32
	if patch.MinReplicas != nil {
		minReplicas = *patch.MinReplicas
	}
	if patch.MaxReplicas != nil {
		maxReplicas = *patch.MaxReplicas
	}

	if maxReplicas < minReplicas {
		replicas := fmt.Sprintf("minReplicas: %d, maxReplicas: %d", minReplicas, maxReplicas)

		*allErrs = append(*allErrs, field.Invalid(path, replicas, "maxReplicas cannot be less than minReplicas"))
	}

}

func ValidateEphemeralStorage(podSpecPatch v1.PodSpec, allErrs *field.ErrorList) {
	path := field.NewPath("spec").
		Child("podSpecPatch").
		Child("containers").
		Key("mapserver").
		Child("resources").
		Child("limits").
		Child(v1.ResourceEphemeralStorage.String())
	storageSet := false
	for _, container := range podSpecPatch.Containers {
		if container.Name == "mapserver" {
			_, storageSet = container.Resources.Limits[v1.ResourceEphemeralStorage]
		}
	}
	if !storageSet {
		*allErrs = append(*allErrs, field.Required(path, ""))
	}
}

func ValidateInspire[O WMSWFS](obj O, allErrs *field.ErrorList, allWarnings *[]string) {
	if obj.Inspire() == nil {
		return
	}

	datasetIDs := obj.DatasetMetadataIDs()
	spatialID := obj.Inspire().SpatialDatasetIdentifier

	if slices.Contains(datasetIDs, spatialID) {
		*allWarnings = append(*allWarnings, field.Invalid(
			field.NewPath("spec").Child("service").Child("inspire").Child("spatialDatasetIdentifier"),
			spatialID,
			"spatialDatasetIdentifier should not also be used as an datasetMetadataUrl.csw.metadataIdentifier",
		).Error())
	}

	if serviceID := obj.Inspire().ServiceMetadataURL.CSW; serviceID != nil {
		if slices.Contains(datasetIDs, serviceID.MetadataIdentifier) {
			*allErrs = append(*allErrs, field.Invalid(
				field.NewPath("spec").Child("service").Child("inspire").Child("csw").Child("metadataIdentifier"),
				serviceID.MetadataIdentifier,
				"serviceMetadataUrl.csw.metadataIdentifier cannot also be used as an datasetMetadataUrl.csw.metadataIdentifier",
			))
		}

		if spatialID == serviceID.MetadataIdentifier {
			*allErrs = append(*allErrs, field.Invalid(
				field.NewPath("spec").Child("service").Child("inspire").Child("csw").Child("metadataIdentifier"),
				serviceID.MetadataIdentifier,
				"serviceMetadataUrl.csw.metadataIdentifier cannot also be used as the spatialDatasetIdentifier",
			))
		}
	}

	if obj.Type() == ServiceTypeWFS && len(datasetIDs) > 1 {
		*allErrs = append(*allErrs, field.Invalid(
			field.NewPath("spec").Child("service").Child("featureTypes[*]").Child("datasetMetadataUrl").Child("csw").Child("metadataIdentifier"),
			datasetIDs,
			"when Inspire, all featureTypes need use the same datasetMetadataUrl.csw.metadataIdentifier",
		))
	}

}

func ValidateOwnerInfo[O WMSWFS](c client.Client, obj O, allErrs *field.ErrorList) {
	ownerInfoRef := obj.OwnerInfoRef()
	ownerInfo := &smoothoperatorv1.OwnerInfo{}
	objectKey := client.ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      ownerInfoRef,
	}
	ctx := context.Background()
	err := c.Get(ctx, objectKey, ownerInfo)
	fieldPath := field.NewPath("spec").Child("service").Child("ownerInfoRef")
	if err != nil {
		*allErrs = append(*allErrs, field.NotFound(fieldPath, ownerInfoRef))
		return
	}

	if ownerInfo.Spec.NamespaceTemplate == nil {
		*allErrs = append(*allErrs, field.Required(fieldPath, "spec.namespaceTemplate missing in "+ownerInfo.Name))
		return
	}

	if ((obj.Inspire() != nil && obj.Inspire().ServiceMetadataURL.CSW != nil) ||
		len(obj.DatasetMetadataIDs()) > 0) &&
		(ownerInfo.Spec.MetadataUrls == nil || ownerInfo.Spec.MetadataUrls.CSW == nil) {
		*allErrs = append(*allErrs, field.Required(fieldPath, "spec.metadataUrls.csw missing in "+ownerInfo.Name))
		return
	}

	switch obj.Type() {
	case ServiceTypeWFS:
		if ownerInfo.Spec.WFS == nil {
			*allErrs = append(*allErrs, field.Required(fieldPath, "spec.WFS missing in "+ownerInfo.Name))
		}
	case ServiceTypeWMS:
		if ownerInfo.Spec.WMS == nil {
			*allErrs = append(*allErrs, field.Required(fieldPath, "spec.WMS missing in "+ownerInfo.Name))
		}
	}

}
