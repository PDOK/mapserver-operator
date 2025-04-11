/*
MIT License

Copyright (c) 2024 Publieke Dienstverlening op de Kaart

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"context"
	"encoding/json"
	"errors"
	pdoknlv2beta1 "github.com/pdok/mapserver-operator/api/v2beta1"
	smoothoperator1 "github.com/pdok/smooth-operator/api/v1"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"golang.org/x/tools/go/packages"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())
	scheme := runtime.NewScheme()

	var err error
	err = pdoknlv2beta1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = pdoknlv3.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = traefikiov1alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = smoothoperator1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	By("bootstrapping test environment")
	traefikCRDPath := must(getTraefikCRDPath())
	ownerInfoCRDPath := must(getOwnerInfoCRDPath())
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDInstallOptions: envtest.CRDInstallOptions{
			Scheme: scheme,
			Paths: []string{
				filepath.Join("..", "..", "config", "crd", "bases", "pdok.nl_wfs.yaml"),
				filepath.Join("..", "..", "config", "crd", "bases", "pdok.nl_wms.yaml"),
				traefikCRDPath,
				ownerInfoCRDPath,
			},
			ErrorIfPathMissing: true,
		},
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}

func getOwnerInfoCRDPath() (string, error) {
	smoothOperatorModule, err := getModule("github.com/pdok/smooth-operator")
	if err != nil {
		return "", err
	}
	if smoothOperatorModule.Dir == "" {
		return "", errors.New("cannot find path for smooth-operator module")
	}
	return filepath.Join(smoothOperatorModule.Dir, "config", "crd", "bases", "pdok.nl_ownerinfo.yaml"), nil
}

func getTraefikCRDPath() (string, error) {
	traefikModule, err := getModule("github.com/traefik/traefik/v3")
	if err != nil {
		return "", err
	}
	if traefikModule.Dir == "" {
		return "", errors.New("cannot find path for traefik module")
	}
	return filepath.Join(traefikModule.Dir, "integration", "fixtures", "k8s", "01-traefik-crd.yml"), nil
}

func getModule(name string) (module *packages.Module, err error) {
	out, err := exec.Command("go", "list", "-json", "-m", name).Output()
	if err != nil {
		return
	}
	module = &packages.Module{}
	err = json.Unmarshal(out, module)
	return
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}
