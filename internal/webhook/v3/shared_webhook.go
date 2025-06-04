package v3

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"sigs.k8s.io/yaml"
)

const (
	samplesPath = "test_data/"
)

func readOwnerInfo(ownerInfo *smoothoperatorv1.OwnerInfo) error {
	data, err := os.ReadFile(samplesPath + "ownerinfo.yaml")
	if err != nil {
		return err
	}
	err = yaml.UnmarshalStrict(data, ownerInfo)
	if err != nil {
		return err
	}
	return err
}

func createOwnerInfo(ctx context.Context, c client.Client, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	clusterOwner := &smoothoperatorv1.OwnerInfo{}
	objectKeyOwner := k8stypes.NamespacedName{
		Namespace: ownerInfo.GetNamespace(),
		Name:      ownerInfo.GetName(),
	}

	err := c.Get(ctx, objectKeyOwner, clusterOwner)
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		resource := ownerInfo.DeepCopy()
		err = c.Create(ctx, resource)
		if err != nil {
			return err
		}
		err = c.Get(ctx, objectKeyOwner, clusterOwner)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateOwnerInfo(ctx context.Context, c client.Client, ownerInfo *smoothoperatorv1.OwnerInfo) error {
	clusterOwner := &smoothoperatorv1.OwnerInfo{}
	objectKeyOwner := k8stypes.NamespacedName{
		Namespace: ownerInfo.GetNamespace(),
		Name:      ownerInfo.GetName(),
	}

	err := c.Get(ctx, objectKeyOwner, clusterOwner)
	if err != nil {
		return err
	}

	ownerInfo.ResourceVersion = clusterOwner.ResourceVersion

	err = c.Update(ctx, ownerInfo)
	if err != nil {
		return err
	}

	return nil
}

func getSampleFilename[W pdoknlv3.WMSWFS](webservice W) (string, error) {
	switch any(webservice).(type) {
	case *pdoknlv3.WFS:
		if _, ok := any(webservice).(*pdoknlv3.WFS); ok {
			return samplesPath + "v3_wfs.yaml", nil
		}
	case *pdoknlv3.WMS:
		if _, ok := any(webservice).(*pdoknlv3.WMS); ok {
			return samplesPath + "v3_wms.yaml", nil
		}
	}
	return "", errors.New("unknown webservice type, cannot determine sample filename")
}

func readSample[W pdoknlv3.WMSWFS](webservice W) error {
	sampleFilename, err := getSampleFilename(webservice)
	if err != nil {
		return err
	}
	sampleYaml, err := os.ReadFile(sampleFilename)
	if err != nil {
		return err
	}
	sampleJSON, err := yaml.YAMLToJSONStrict(sampleYaml)
	if err != nil {
		return err
	}
	err = json.Unmarshal(sampleJSON, webservice)
	if err != nil {
		return err
	}

	return nil
}
