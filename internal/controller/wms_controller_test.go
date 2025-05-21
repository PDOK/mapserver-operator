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

//nolint:revive // Complains about the dot imports
import (
	"context"
	_ "embed"
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller/mapserver"
	"github.com/pdok/mapserver-operator/internal/controller/utils"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorsamples "github.com/pdok/smooth-operator/config/samples"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	smoothoperatorvalidation "github.com/pdok/smooth-operator/pkg/validation"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

const (
	wmsResourceName = "wms-resource"
)

var _ = Describe("WMS Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		// Setup variables for unique WMS resource per It node
		counter := 1
		var typeNamespacedNameWms types.NamespacedName

		wms := &pdoknlv3.WMS{}

		typeNamespacedNameOwnerInfo := types.NamespacedName{
			Namespace: namespace,
			Name:      ownerInfoResourceName,
		}
		ownerInfo := &smoothoperatorv1.OwnerInfo{}
		initScripts, mapfileGenerator := true, true

		BeforeEach(func() {
			pdoknlv3.SetHost("localhost")

			// Create a unique WMS resource for every It node to prevent unexpected resource state caused by finalizers
			sampleWms, err := getUniqueWMSSample(counter)
			Expect(err).To(BeNil())
			typeNamespacedNameWms = getUniqueWmsTypeNamespacedName(counter)
			counter++

			By("creating the custom resource for the Kind WMS")
			err = k8sClient.Get(ctx, typeNamespacedNameWms, wms)
			if err != nil && apierrors.IsNotFound(err) {
				resource := sampleWms.DeepCopy()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedNameWms, wms)).To(Succeed())
			}

			By("creating the custom resource for the Kind OwnerInfo")
			ownerInfo, err = smoothoperatorsamples.OwnerInfoSample()
			ownerInfo.Namespace = namespace
			Expect(err).To(BeNil())
			err = k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfo)
			if err != nil && apierrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, ownerInfo)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfo)).To(Succeed())
			}
		})

		AfterEach(func() {
			wmsResource := &pdoknlv3.WMS{}
			wmsResource.Namespace = namespace
			wmsResource.Name = typeNamespacedNameWms.Name
			err := k8sClient.Get(ctx, typeNamespacedNameWms, wmsResource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WMS")
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, wmsResource))).To(Succeed())

			ownerInfoResource := &smoothoperatorv1.OwnerInfo{}
			err = k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfoResource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance OwnerInfo")
			Expect(k8sClient.Delete(ctx, ownerInfoResource)).To(Succeed())

			// the testEnv does not do garbage collection (https://book.kubebuilder.io/reference/envtest#testing-considerations)
			By("Cleaning Owned Resources")
			objects, err := getExpectedObjects(ctx, wms, initScripts, mapfileGenerator)
			Expect(err).NotTo(HaveOccurred())
			for _, o := range objects {
				objectKey := client.ObjectKey{
					Namespace: o.GetNamespace(),
					Name:      o.GetName(),
				}
				err := k8sClient.Get(ctx, objectKey, o)
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Delete(ctx, o)).To(Succeed())
			}
		})

		It("Should successfully reconcile the resource", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the created resource")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			By("Waiting for the owned resources to be created")
			initScripts = false
			expectedBareObjects, err := getExpectedObjects(ctx, wms, initScripts, mapfileGenerator)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() error {
				for _, o := range expectedBareObjects {
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: o.GetName()}, o)
					if err != nil {
						return err
					}
				}
				return nil
			}, "10s", "1s").Should(Not(HaveOccurred()))

			By("Checking the status of the WMS")
			err = k8sClient.Get(ctx, typeNamespacedNameWms, wms)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(wms.Status.Conditions)).To(BeEquivalentTo(1))
			Expect(wms.Status.Conditions[0].Status).To(BeEquivalentTo(metav1.ConditionTrue))
		})

		It("Should successfully reconcile after a change in an owned resource", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS, checking the finalizer, and reconciling again")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			By("Getting the original Deployment")
			deployment := getBareDeployment(wms)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())
			originalRevisionHistoryLimit := *deployment.Spec.RevisionHistoryLimit

			By("Altering the Deployment")
			err := k8sClient.Patch(ctx, deployment, client.RawPatch(types.MergePatchType, []byte(
				`{"spec": {"revisionHistoryLimit": 99}}`)))
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was altered")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(*deployment.Spec.RevisionHistoryLimit).To(BeEquivalentTo(99))
			}, "10s", "1s").Should(BeTrue())

			By("Reconciling the WMS again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWms})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was restored")
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(*deployment.Spec.RevisionHistoryLimit).To(BeEquivalentTo(originalRevisionHistoryLimit))
			}, "10s", "1s").Should(BeTrue())
		})

		It("Should create correct deployment manifest.", func() {
			controllerReconciler := getWMSReconciler()
			reconcilerImages := getReconcilerImages(controllerReconciler)

			By("Reconciling the WMS and checking the deployment")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			deployment := getBareDeployment(wms)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(deployment.GetName()).Should(Equal(wms.GetName() + "-wms-mapserver"))
			Expect(deployment.GetNamespace()).Should(Equal(namespace))

			Expect(deployment.Spec.Template.Spec.TerminationGracePeriodSeconds).Should(Equal(smoothoperatorutils.Pointer(int64(60))))

			/**
			Label + selector tests
			*/
			checkWMSLabels(deployment.GetLabels(), deployment.Spec.Selector.MatchLabels)

			/**
			Container tests
			*/
			containerMapserver := deployment.Spec.Template.Spec.Containers[0]
			Expect(containerMapserver.Name).Should(Equal("mapserver"))
			Expect(containerMapserver.Ports[0].ContainerPort).Should(Equal(int32(80)))
			Expect(containerMapserver.Image).Should(Equal(reconcilerImages.MapserverImage))
			Expect(containerMapserver.ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
			Expect(containerMapserver.Resources.Limits.Memory().String()).Should(Equal("12M"))
			Expect(containerMapserver.Resources.Requests.Cpu().String()).Should(Equal("100m"))
			Expect(len(containerMapserver.LivenessProbe.Exec.Command)).Should(Equal(3))
			Expect(containerMapserver.LivenessProbe.Exec.Command[2]).Should(Equal("wget -SO- -T 10 -t 2 'http://127.0.0.1:80/mapserver?SERVICE=WMS&request=GetCapabilities' 2>&1 | egrep -aiA10 'HTTP/1.1 200' | egrep -i 'Content-Type: text/xml'"))
			Expect(containerMapserver.LivenessProbe.FailureThreshold).Should(Equal(int32(3)))
			Expect(containerMapserver.LivenessProbe.InitialDelaySeconds).Should(Equal(int32(20)))
			Expect(containerMapserver.LivenessProbe.PeriodSeconds).Should(Equal(int32(10)))
			Expect(containerMapserver.LivenessProbe.TimeoutSeconds).Should(Equal(int32(10)))
			Expect(len(containerMapserver.ReadinessProbe.Exec.Command)).Should(Equal(3))
			Expect(containerMapserver.ReadinessProbe.Exec.Command[2]).Should(Equal("wget -SO- -T 10 -t 2 'http://127.0.0.1:80/mapserver?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&BBOX=190061.4619730016857,462435.5987861062749,202917.7508707302331,473761.6884966178914&CRS=EPSG:28992&WIDTH=100&HEIGHT=100&LAYERS=gpkg-layer-name&STYLES=&FORMAT=image/png' 2>&1 | egrep -aiA10 'HTTP/1.1 200' | egrep -i 'Content-Type: image/png'"))
			Expect(containerMapserver.ReadinessProbe.FailureThreshold).Should(Equal(int32(3)))
			Expect(containerMapserver.ReadinessProbe.InitialDelaySeconds).Should(Equal(int32(20)))
			Expect(containerMapserver.ReadinessProbe.PeriodSeconds).Should(Equal(int32(10)))
			Expect(containerMapserver.ReadinessProbe.TimeoutSeconds).Should(Equal(int32(10)))
			Expect(len(containerMapserver.StartupProbe.Exec.Command)).Should(Equal(3))
			Expect(containerMapserver.StartupProbe.Exec.Command[2]).Should(Equal("wget -SO- -T 10 -t 2 'http://127.0.0.1:80/mapserver?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&BBOX=190061.4619730016857,462435.5987861062749,202917.7508707302331,473761.6884966178914&CRS=EPSG:28992&WIDTH=100&HEIGHT=100&LAYERS=top-layer-name,group-layer-name,gpkg-layer-name,tif-layer-name&STYLES=&FORMAT=image/png' 2>&1 | egrep -aiA10 'HTTP/1.1 200' | egrep -i 'Content-Type: image/png'"))
			Expect(containerMapserver.StartupProbe.FailureThreshold).Should(Equal(int32(3)))
			Expect(containerMapserver.StartupProbe.InitialDelaySeconds).Should(Equal(int32(20)))
			Expect(containerMapserver.StartupProbe.PeriodSeconds).Should(Equal(int32(10)))
			Expect(containerMapserver.StartupProbe.TimeoutSeconds).Should(Equal(int32(10)))

			containerOgcWebserviceProxy := deployment.Spec.Template.Spec.Containers[2]
			Expect(containerOgcWebserviceProxy.Name).Should(Equal(OgcWebserviceProxyName))
			ogcWebserviceProxyCommands := []string{"/ogc-webservice-proxy", "-h=http://127.0.0.1/", "-t=wms", "-s=/input/service-config.yaml", "-v", "-d=15"}
			Expect(containerOgcWebserviceProxy.Command).Should(Equal(ogcWebserviceProxyCommands))
			Expect(containerOgcWebserviceProxy.Image).Should(Equal(reconcilerImages.OgcWebserviceProxyImage))
			Expect(containerOgcWebserviceProxy.ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
			Expect(containerOgcWebserviceProxy.Ports[0].ContainerPort).Should(Equal(int32(9111)))
			Expect(containerOgcWebserviceProxy.Resources.Limits.Memory().String()).Should(Equal("200M"))
			Expect(containerOgcWebserviceProxy.Resources.Requests.Cpu().String()).Should(Equal("50m"))
			volumeMountsContainerOgcWebserviceProxy := []corev1.VolumeMount{
				{Name: mapserver.ConfigMapOgcWebserviceProxyVolumeName, MountPath: "/input", ReadOnly: true},
			}
			Expect(containerOgcWebserviceProxy.VolumeMounts).Should(Equal(volumeMountsContainerOgcWebserviceProxy))

			/**
			Init container tests
			*/

			blobDownloadContainer, err := getInitContainer("blob-download", deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobDownloadContainer.Image).Should(Equal(reconcilerImages.MultitoolImage))
			volumeMounts := []corev1.VolumeMount{
				{Name: "base", MountPath: "/srv/data"},
				{Name: "data", MountPath: "/var/www"},
				{Name: mapserver.ConfigMapBlobDownloadVolumeName, MountPath: "/srv/scripts", ReadOnly: true},
			}
			envFrom := []corev1.EnvFromSource{
				utils.NewEnvFromSource(utils.EnvFromSourceTypeConfigMap, "blobs-testtest"),
				utils.NewEnvFromSource(utils.EnvFromSourceTypeSecret, "blobs-testtest"),
			}
			Expect(blobDownloadContainer.VolumeMounts).Should(Equal(volumeMounts))
			Expect(blobDownloadContainer.EnvFrom).Should(Equal(envFrom))
			Expect(blobDownloadContainer.Command).Should(Equal([]string{"/bin/sh", "-c"}))
			Expect(len(blobDownloadContainer.Args)).Should(BeNumerically("==", 1))

			mapfileGeneratorContainer, err := getInitContainer("mapfile-generator", deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(mapfileGeneratorContainer.Image).Should(Equal(reconcilerImages.MapfileGeneratorImage))
			volumeMounts = []corev1.VolumeMount{
				{Name: "base", MountPath: "/srv/data"},
				{Name: mapserver.ConfigMapMapfileGeneratorVolumeName, MountPath: "/input", ReadOnly: true},
				{Name: mapserver.ConfigMapStylingFilesVolumeName, MountPath: "/styling", ReadOnly: true},
			}
			Expect(mapfileGeneratorContainer.VolumeMounts).Should(Equal(volumeMounts))
			Expect(mapfileGeneratorContainer.Command).Should(Equal([]string{"generate-mapfile"}))
			Expect(mapfileGeneratorContainer.Args).Should(Equal([]string{"--not-include", "wms", "/input/input.json", "/srv/data/config/mapfile"}))

			capabilitiesGeneratorContainer, err := getInitContainer("capabilities-generator", deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(capabilitiesGeneratorContainer.Image).Should(Equal(reconcilerImages.CapabilitiesGeneratorImage))
			Expect(capabilitiesGeneratorContainer.ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))

			volumeMounts = []corev1.VolumeMount{
				{Name: "data", MountPath: "/var/www"},
				{Name: mapserver.ConfigMapCapabilitiesGeneratorVolumeName, MountPath: "/input", ReadOnly: true},
			}
			Expect(capabilitiesGeneratorContainer.VolumeMounts).Should(Equal(volumeMounts))
			env := []corev1.EnvVar{
				{Name: "SERVICECONFIG", Value: "/input/input.yaml"},
			}
			Expect(capabilitiesGeneratorContainer.Env).Should(Equal(env))

			featureinfoGeneratorContainer, err := getInitContainer(FeatureInfoGeneratorName, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(featureinfoGeneratorContainer.Image).Should(Equal(reconcilerImages.FeatureinfoGeneratorImage))
			Expect(featureinfoGeneratorContainer.ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
			Expect(featureinfoGeneratorContainer.Command).Should(Equal([]string{FeatureInfoGeneratorName}))
			volumeMounts = []corev1.VolumeMount{
				{Name: "base", MountPath: "/srv/data", ReadOnly: false},
				{Name: mapserver.ConfigMapFeatureinfoGeneratorVolumeName, MountPath: "/input", ReadOnly: true},
			}
			Expect(featureinfoGeneratorContainer.VolumeMounts).Should(Equal(volumeMounts))
			Expect(len(featureinfoGeneratorContainer.Args)).Should(BeNumerically("==", 6))

			legendGeneratorContainer, err := getInitContainer(LegendGeneratorName, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(legendGeneratorContainer.Image).Should(Equal(reconcilerImages.MapserverImage))
			Expect(legendGeneratorContainer.ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
			Expect(len(legendGeneratorContainer.Command)).Should(BeNumerically("==", 3))
			volumeMounts = []corev1.VolumeMount{
				{Name: "base", MountPath: "/srv/data", ReadOnly: false},
				{Name: "data", MountPath: "/var/www", ReadOnly: false},
				{Name: "mapserver", MountPath: "/srv/mapserver/config/default_mapserver.conf", SubPath: "default_mapserver.conf"},
				{Name: mapserver.ConfigMapLegendGeneratorVolumeName, MountPath: "/input", ReadOnly: true},
			}
			Expect(legendGeneratorContainer.VolumeMounts).Should(Equal(volumeMounts))

			env = []corev1.EnvVar{
				{Name: "MAPSERVER_CONFIG_FILE", Value: "/srv/mapserver/config/default_mapserver.conf", ValueFrom: nil},
				{Name: "MS_MAPFILE", Value: "/srv/data/config/mapfile/service.map", ValueFrom: nil},
			}
			Expect(legendGeneratorContainer.Env).Should(Equal(env))

			/**
			Volumes tests
			*/
			expectedVolumes := []string{"" +
				"base",
				"data",
				mapserver.ConfigMapVolumeName,
				mapserver.ConfigMapBlobDownloadVolumeName,
				mapserver.ConfigMapCapabilitiesGeneratorVolumeName,
				mapserver.ConfigMapMapfileGeneratorVolumeName,
				mapserver.ConfigMapOgcWebserviceProxyVolumeName,
				mapserver.ConfigMapLegendGeneratorVolumeName,
				mapserver.ConfigMapFeatureinfoGeneratorVolumeName,
			}
			for _, ev := range expectedVolumes {
				Expect(slices.IndexFunc(deployment.Spec.Template.Spec.Volumes, func(v corev1.Volume) bool {
					return v.Name == ev
				})).ShouldNot(BeEquivalentTo(-1))
			}

		})

		It("Should not mount a blob download configmap if options.prefetchData is false.", func() {
			wmsResource := &pdoknlv3.WMS{}
			wmsResource.Namespace = namespace
			wmsResource.Name = typeNamespacedNameWms.Name
			err := k8sClient.Get(ctx, typeNamespacedNameWms, wmsResource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WMS")
			Expect(k8sClient.Delete(ctx, wmsResource)).To(Succeed())

			sampleWms, err := getUniqueWMSSample(counter)
			counter++
			typeNamespacedNameWms.Name = sampleWms.Name
			Expect(err).NotTo(HaveOccurred())
			sampleWms.Spec.Options.PrefetchData = false

			Expect(k8sClient.Create(ctx, sampleWms.DeepCopy())).To(Succeed())
			Expect(k8sClient.Get(ctx, typeNamespacedNameWms, wms)).To(Succeed())

			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			_, err = getHashedConfigMapNameFromClient(ctx, wms, mapserver.ConfigMapBlobDownloadVolumeName)
			Expect(err).To(HaveOccurred())
		})

		It("Should not mount a ogc web service proxy configmap if options.DisableWebserviceProxy is true.", func() {
			wmsResource := &pdoknlv3.WMS{}
			wmsResource.Namespace = namespace
			wmsResource.Name = typeNamespacedNameWms.Name
			err := k8sClient.Get(ctx, typeNamespacedNameWms, wmsResource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WMS")
			Expect(k8sClient.Delete(ctx, wmsResource)).To(Succeed())

			sampleWms, err := getUniqueWMSSample(counter)
			counter++
			typeNamespacedNameWms.Name = sampleWms.Name
			Expect(err).NotTo(HaveOccurred())
			sampleWms.Spec.Options.DisableWebserviceProxy = true
			Expect(k8sClient.Create(ctx, sampleWms.DeepCopy())).To(Succeed())
			Expect(k8sClient.Get(ctx, typeNamespacedNameWms, wms)).To(Succeed())

			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			_, err = getHashedConfigMapNameFromClient(ctx, wms, mapserver.ConfigMapOgcWebserviceProxyVolumeName)
			Expect(err).To(HaveOccurred())
		})

		It("Should mount a legend-fixer initcontainer if options.REWRITEGROUPTODATALAYERS is TRUE.", func() {
			wmsResource := &pdoknlv3.WMS{}
			wmsResource.Namespace = namespace
			wmsResource.Name = typeNamespacedNameWms.Name
			err := k8sClient.Get(ctx, typeNamespacedNameWms, wmsResource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WMS")
			Expect(k8sClient.Delete(ctx, wmsResource)).To(Succeed())

			sampleWms, err := getUniqueWMSSample(counter)
			counter++
			typeNamespacedNameWms.Name = sampleWms.Name
			Expect(err).NotTo(HaveOccurred())
			sampleWms.Spec.Options.RewriteGroupToDataLayers = true

			Expect(k8sClient.Create(ctx, sampleWms.DeepCopy())).To(Succeed())
			Expect(k8sClient.Get(ctx, typeNamespacedNameWms, wms)).To(Succeed())

			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)
			deployment := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: getBareDeployment(wms).GetName()}, deployment)
			Expect(err).NotTo(HaveOccurred())
			blobDownloadContainer, _ := getInitContainer("blob-download", deployment)
			Expect(blobDownloadContainer.Name).To(BeEquivalentTo("blob-download"))
			legendFixerContainer, err := getInitContainer("legend-fixer", deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(legendFixerContainer.Image).Should(Equal(controllerReconciler.Images.MultitoolImage))
			Expect(legendFixerContainer.ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
			Expect(len(legendFixerContainer.Command)).Should(BeNumerically("==", 2))

			volumeMounts := []corev1.VolumeMount{
				{Name: "data", MountPath: "/var/www", ReadOnly: false},
				{Name: mapserver.ConfigMapLegendGeneratorVolumeName, MountPath: "/input", ReadOnly: true},
			}
			Expect(legendFixerContainer.VolumeMounts).Should(Equal(volumeMounts))

			configMap := getBareConfigMap(wms, LegendGeneratorName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wms, mapserver.ConfigMapLegendGeneratorVolumeName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wms.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(configMap.GetName()).To(HavePrefix(wms.GetName() + "-wms-legend-generator-"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))
			checkWMSLabels(configMap.GetLabels())

			data, ok := configMap.Data["legend-fixer.sh"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))

			_, ok = configMap.Data["remove"]
			Expect(ok).To(BeTrue())

			data, ok = configMap.Data["ogc-webservice-proxy-config.yaml"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))

			// actual configMap content is tested in legendgenerator/legend_generator_test.go
		})

		It("ogcWebserviceProxyCommands will not contain command '-v' if options.VALIDATEREQUESTS is FALSE.", func() {
			wmsResource := &pdoknlv3.WMS{}
			wmsResource.Namespace = namespace
			wmsResource.Name = typeNamespacedNameWms.Name
			err := k8sClient.Get(ctx, typeNamespacedNameWms, wmsResource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WMS")
			Expect(k8sClient.Delete(ctx, wmsResource)).To(Succeed())

			sampleWms, err := getUniqueWMSSample(counter)
			counter++
			typeNamespacedNameWms.Name = sampleWms.Name
			Expect(err).NotTo(HaveOccurred())
			sampleWms.Spec.Options.ValidateRequests = false

			Expect(k8sClient.Create(ctx, sampleWms.DeepCopy())).To(Succeed())
			Expect(k8sClient.Get(ctx, typeNamespacedNameWms, wms)).To(Succeed())

			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)
			deployment := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: getBareDeployment(wms).GetName()}, deployment)
			Expect(err).NotTo(HaveOccurred())

			containerOgcWebserviceProxy := deployment.Spec.Template.Spec.Containers[2]
			Expect(containerOgcWebserviceProxy.Name).Should(Equal(OgcWebserviceProxyName))
			ogcWebserviceProxyCommands := []string{"/ogc-webservice-proxy", "-h=http://127.0.0.1/", "-t=wms", "-s=/input/service-config.yaml", "-d=15"}
			Expect(containerOgcWebserviceProxy.Command).Should(Equal(ogcWebserviceProxyCommands))
			Expect(containerOgcWebserviceProxy.ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
			Expect(containerOgcWebserviceProxy.Ports[0].ContainerPort).Should(Equal(int32(9111)))
			Expect(containerOgcWebserviceProxy.Resources.Limits.Memory().String()).Should(Equal("200M"))
			Expect(containerOgcWebserviceProxy.Resources.Requests.Cpu().String()).Should(Equal("50m"))
			volumeMountsContainerOgcWebserviceProxy := []corev1.VolumeMount{
				{Name: mapserver.ConfigMapOgcWebserviceProxyVolumeName, MountPath: "/input", ReadOnly: true},
			}
			Expect(containerOgcWebserviceProxy.VolumeMounts).Should(Equal(volumeMountsContainerOgcWebserviceProxy))

		})

		It("Should create correct configMap manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			configMap := getBareConfigMap(wms, MapserverName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wms, mapserver.ConfigMapVolumeName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wms.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			// Make sure the name is hashed
			Expect(configMap.GetName()).To(HavePrefix(wms.GetName() + "-wms-mapserver-"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))

			checkWMSLabels(configMap.GetLabels())

			defaultMapserverConf, ok := configMap.Data["default_mapserver.conf"]
			Expect(ok).To(BeTrue())
			Expect(defaultMapserverConf).To(ContainSubstring("MAP \"/srv/data/config/mapfile/service.map\""))

			includeConf, ok := configMap.Data["include.conf"]
			Expect(ok).To(BeTrue())
			Expect(includeConf).To(ContainSubstring("/owner/dataset/wms"))

			ogcLua, ok := configMap.Data["ogc.lua"]
			Expect(ok).To(BeTrue())
			Expect(ogcLua).To(ContainSubstring("/srv/mapserver/config/scraping-error.xml"))

			scrapingErrorXML, ok := configMap.Data["scraping-error.xml"]
			Expect(ok).To(BeTrue())
			Expect(scrapingErrorXML).To(ContainSubstring("It is not possible to use a 'startindex' higher than 50.000"))
		})

		It("Should create correct configMapMapfileGenerator manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			configMap := getBareConfigMap(wms, mapfileGeneratorInput)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wms, mapserver.ConfigMapMapfileGeneratorVolumeName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wms.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(configMap.GetName()).To(HavePrefix(wms.GetName() + "-wms-mapfile-generator-"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))
			checkWMSLabels(configMap.GetLabels())

			data, ok := configMap.Data["input.json"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))
			// input.json content is tested in mapfilegenerator/mapfile_generator_test.go
		})

		It("Should create correct configMapBlobDownload manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			configMap := getBareConfigMap(wms, InitScriptsName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wms, mapserver.ConfigMapBlobDownloadVolumeName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wms.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(configMap.GetName()).To(HavePrefix(wms.GetName() + "-wms-init-scripts-"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))
			checkWMSLabels(configMap.GetLabels())

			data, ok := configMap.Data["gpkg_download.sh"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))
			// gpkg_download.sh content is tested in blobdownload/blob_download_test.go
		})

		It("Should create correct configMapCapabilitiesGenerator manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			configMap := getBareConfigMap(wms, CapabilitiesGeneratorName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wms, mapserver.ConfigMapCapabilitiesGeneratorVolumeName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wms.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(configMap.GetName()).To(HavePrefix(wms.GetName() + "-wms-capabilities-generator-"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))
			checkWMSLabels(configMap.GetLabels())

			data, ok := configMap.Data["input.yaml"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))
			// input.yaml content is tested in capabilitiesgenerator/capabilities_generator_test.go
		})

		It("Should create correct configMapLegendGenerator manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			configMap := getBareConfigMap(wms, LegendGeneratorName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wms, mapserver.ConfigMapLegendGeneratorVolumeName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wms.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(configMap.GetName()).To(HavePrefix(wms.GetName() + "-wms-legend-generator-"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))
			checkWMSLabels(configMap.GetLabels())

			data, ok := configMap.Data["default_mapserver.conf"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))

			_, ok = configMap.Data["input"]
			Expect(ok).To(BeTrue())
		})

		It("Should create correct configMapFeatureinfoGenerator manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			configMap := getBareConfigMap(wms, FeatureInfoGeneratorName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wms, mapserver.ConfigMapFeatureinfoGeneratorVolumeName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wms.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(configMap.GetName()).To(HavePrefix(wms.GetName() + "-wms-featureinfo-generator"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))
			checkWMSLabels(configMap.GetLabels())

			data, ok := configMap.Data["input.json"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))

			// input.json content is tested in featureinfogenerator/featureinfo_generator_test.go
		})

		It("Should create correct middlewareCorsHeaders manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the middlewareCorsHeaders")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			middlewareCorsHeaders := getBareCorsHeadersMiddleware(wms)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(middlewareCorsHeaders), middlewareCorsHeaders)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(middlewareCorsHeaders.Name).Should(Equal(wms.GetName() + "-wms-mapserver-headers"))
			Expect(middlewareCorsHeaders.Namespace).Should(Equal("default"))
			checkWMSLabels(middlewareCorsHeaders.GetLabels())
			// Expect(middlewareCorsHeaders.Spec.Headers.FrameDeny).Should(Equal(true))
			Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Headers"]).Should(Equal("Content-Type"))
			Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Method"]).Should(Equal("GET, POST, OPTIONS"))
			Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Origin"]).Should(Equal("*"))
		})

		It("Should create correct podDisruptionBudget manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the podDisruptionBudget")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			podDisruptionBudget := getBarePodDisruptionBudget(wms)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			/**
			Label + selector tests
			*/
			checkWMSLabels(podDisruptionBudget.GetLabels(), podDisruptionBudget.Spec.Selector.MatchLabels)

			Expect(podDisruptionBudget.GetName()).To(Equal(wms.GetName() + "-wms-mapserver"))
			Expect(podDisruptionBudget.Spec.MaxUnavailable.IntValue()).Should(Equal(1))
		})

		It("Should create correct horizontalPodAutoScaler manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the horizontalPodAutoScaler")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			autoscaler := getBareHorizontalPodAutoScaler(wms)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(autoscaler), autoscaler)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(autoscaler.GetName()).To(Equal(wms.GetName() + "-wms-mapserver"))
			Expect(autoscaler.Spec.ScaleTargetRef).To(Equal(autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       wms.GetName() + "-wms-mapserver",
			}))

			/**
			Label + selector tests
			*/
			checkWMSLabels(autoscaler.GetLabels())

			Expect(autoscaler.Spec.MinReplicas).Should(Equal(smoothoperatorutils.Pointer(int32(2))))
			Expect(autoscaler.Spec.MaxReplicas).Should(Equal(int32(5)))
			Expect(autoscaler.Spec.Behavior).ToNot(BeNil())
			Expect(autoscaler.Spec.Behavior.ScaleDown).ToNot(BeNil())
			Expect(autoscaler.Spec.Behavior.ScaleUp).ToNot(BeNil())
			Expect(autoscaler.Spec.Behavior.ScaleDown).To(Equal(&autoscalingv2.HPAScalingRules{
				StabilizationWindowSeconds: smoothoperatorutils.Pointer(int32(3600)),
				SelectPolicy:               smoothoperatorutils.Pointer(autoscalingv2.MaxChangePolicySelect),
				Policies: []autoscalingv2.HPAScalingPolicy{
					{
						PeriodSeconds: int32(600),
						Value:         int32(10),
						Type:          autoscalingv2.PercentScalingPolicy,
					},
					{
						PeriodSeconds: int32(600),
						Value:         int32(1),
						Type:          autoscalingv2.PodsScalingPolicy,
					},
				},
			}))
			Expect(autoscaler.Spec.Behavior.ScaleUp).To(Equal(&autoscalingv2.HPAScalingRules{
				StabilizationWindowSeconds: smoothoperatorutils.Pointer(int32(0)),
				SelectPolicy:               smoothoperatorutils.Pointer(autoscalingv2.MaxChangePolicySelect),
				Policies: []autoscalingv2.HPAScalingPolicy{
					{
						PeriodSeconds: int32(60),
						Value:         int32(20),
						Type:          autoscalingv2.PodsScalingPolicy,
					},
				},
			}))
			Expect(autoscaler.Spec.Metrics).To(Equal([]autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: smoothoperatorutils.Pointer(int32(120)),
						},
					},
				},
			}))
		})

		It("Should create correct service manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the service")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			service := getBareService(wms)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(service), service)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(service.GetName()).To(Equal(wms.GetName() + "-wms-mapserver"))
			Expect(service.Spec.Ports).To(Equal([]corev1.ServicePort{
				{
					Name:       "mapserver",
					Port:       80,
					TargetPort: intstr.FromInt32(80),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       OgcWebserviceProxyName,
					Port:       9111,
					TargetPort: intstr.FromInt32(9111),
					Protocol:   "TCP",
				},
				{
					Name:       "metric",
					Port:       9117,
					TargetPort: intstr.FromInt32(9117),
					Protocol:   corev1.ProtocolTCP,
				},
			}))

			/**
			Label + selector tests
			*/
			checkWMSLabels(service.GetLabels(), service.Spec.Selector)
		})

		It("Should create correct ingressRoute manifest.", func() {
			controllerReconciler := getWMSReconciler()

			By("Reconciling the WMS and checking the ingressRoute")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)

			ingressRoute := getBareIngressRoute(wms)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ingressRoute), ingressRoute)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			checkWMSLabels(ingressRoute.GetLabels())

			Expect(ingressRoute.Annotations).To(Equal(map[string]string{
				"uptime.pdok.nl/id":   wms.ID(),
				"uptime.pdok.nl/name": "OWNER dataset 1.0.0 INSPIRE WMS",
				"uptime.pdok.nl/tags": "public-stats,wms,inspire",
				"uptime.pdok.nl/url":  "https://service.pdok.nl/owner/dataset/wms/1.0.0",
			}))

			Expect(ingressRoute.GetName()).To(Equal(wms.GetName() + "-wms-mapserver"))
			Expect(len(ingressRoute.Spec.Routes)).To(Equal(2))
			Expect(ingressRoute.Spec.Routes[0]).To(Equal(traefikiov1alpha1.Route{
				Kind:        "Rule",
				Match:       "Host(`localhost`) && PathPrefix(`/owner/dataset/wms/1.0.0/legend`)",
				Middlewares: []traefikiov1alpha1.MiddlewareRef{{Name: wms.GetName() + "-wms-mapserver-headers"}},
				Services: []traefikiov1alpha1.Service{{
					LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
						Kind: "Service",
						Port: intstr.FromInt32(80),
						Name: wms.GetName() + "-wms-mapserver",
					},
				}},
			}))
			Expect(ingressRoute.Spec.Routes[1]).To(Equal(traefikiov1alpha1.Route{
				Kind:        "Rule",
				Match:       "Host(`localhost`) && Path(`/owner/dataset/wms/1.0.0`)",
				Middlewares: []traefikiov1alpha1.MiddlewareRef{{Name: wms.GetName() + "-wms-mapserver-headers"}},
				Services: []traefikiov1alpha1.Service{{
					LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
						Kind: "Service",
						Port: intstr.FromInt32(9111),
						Name: wms.GetName() + "-wms-mapserver",
					},
				}},
			}))
		})

		It("Static mapfile options.", func() {
			wmsResource := &pdoknlv3.WMS{}
			wmsResource.Namespace = namespace
			wmsResource.Name = typeNamespacedNameWms.Name
			err := k8sClient.Get(ctx, typeNamespacedNameWms, wmsResource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WMS")
			Expect(k8sClient.Delete(ctx, wmsResource)).To(Succeed())

			sampleWms, err := getUniqueWMSSample(counter)
			counter++
			sampleWms.Spec.Service.Mapfile = &pdoknlv3.Mapfile{ConfigMapKeyRef: corev1.ConfigMapKeySelector{Key: "mapfile.map", LocalObjectReference: corev1.LocalObjectReference{Name: "custom-mapfile"}}}
			typeNamespacedNameWms.Name = sampleWms.Name

			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Create(ctx, sampleWms.DeepCopy())).To(Succeed())
			Expect(k8sClient.Get(ctx, typeNamespacedNameWms, wms)).To(Succeed())

			controllerReconciler := getWMSReconciler()
			mapfileGenerator = false
			By("Reconciling the WMS and checking the configMap")
			reconcileWMS(controllerReconciler, wms, typeNamespacedNameWms)
			deployment := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: getBareDeployment(wms).GetName()}, deployment)
			Expect(err).NotTo(HaveOccurred())

			volumeMounts := []corev1.VolumeMount{
				{Name: "base", MountPath: "/srv/data", ReadOnly: false},
				{Name: "data", MountPath: "/var/www", ReadOnly: false},
				{Name: "mapserver", MountPath: "/srv/mapserver/config/default_mapserver.conf", SubPath: "default_mapserver.conf"},
				{Name: "mapfile", MountPath: "/srv/data/config/mapfile"},
				{Name: mapserver.ConfigMapLegendGeneratorVolumeName, MountPath: "/input", ReadOnly: true},
			}
			legendGeneratorContainer, _ := getInitContainer(LegendGeneratorName, deployment)

			Expect(legendGeneratorContainer.VolumeMounts).Should(Equal(volumeMounts))

			env := []corev1.EnvVar{
				{Name: "MAPSERVER_CONFIG_FILE", Value: "/srv/mapserver/config/default_mapserver.conf", ValueFrom: nil},
				{Name: "MS_MAPFILE", Value: "/srv/data/config/mapfile/mapfile.map", ValueFrom: nil},
			}
			Expect(legendGeneratorContainer.Env).Should(Equal(env))
			envContainer := []corev1.EnvVar{
				{Name: "SERVICE_TYPE", Value: "WMS", ValueFrom: nil},
				{
					Name:      "MAPSERVER_CONFIG_FILE",
					Value:     "/srv/mapserver/config/default_mapserver.conf",
					ValueFrom: nil,
				},
				{Name: "MS_MAPFILE", Value: "/srv/data/config/mapfile/mapfile.map", ValueFrom: nil},
				{
					Name:  "AZURE_STORAGE_CONNECTION_STRING",
					Value: "",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "blobs-testtest",
							},
							Key: "AZURE_STORAGE_CONNECTION_STRING",
						},
					},
				},
			}
			Expect(deployment.Spec.Template.Spec.Containers[0].Env).Should(Equal(envContainer))
		})
	})
})

func reconcileWMS(r *WMSReconciler, wms *pdoknlv3.WMS, typeNamespacedNameWms types.NamespacedName) {
	// Reconcile the WMS
	_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWms})
	Expect(err).NotTo(HaveOccurred())

	// Check it's there
	err = k8sClient.Get(ctx, typeNamespacedNameWms, wms)
	Expect(err).NotTo(HaveOccurred())
}

//go:embed test_manifests/v3_wms.yaml
var testManifestWMS []byte

func getUniqueWMSSample(counter int) (*pdoknlv3.WMS, error) {
	sample := &pdoknlv3.WMS{}
	defaulted, err := smoothoperatorvalidation.ApplySchemaDefaultsStr(string(testManifestWMS))
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal([]byte(defaulted), sample)
	if err != nil {
		return nil, err
	}

	sample.Name = getUniqueWmsResourceName(counter)
	sample.Namespace = namespace
	sample.Spec.Service.OwnerInfoRef = ownerInfoResourceName

	return sample, nil
}

func getUniqueWmsTypeNamespacedName(counter int) types.NamespacedName {
	return types.NamespacedName{
		Name:      getUniqueWmsResourceName(counter),
		Namespace: namespace,
	}
}

func getUniqueWmsResourceName(counter int) string {
	return fmt.Sprintf("%s-%v", wmsResourceName, counter)
}

func getWMSReconciler() *WMSReconciler {
	return &WMSReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
		Images: Images{
			MultitoolImage:             testImageName1,
			MapfileGeneratorImage:      testImageName2,
			MapserverImage:             testImageName3,
			CapabilitiesGeneratorImage: testImageName4,
			FeatureinfoGeneratorImage:  testImageName5,
			OgcWebserviceProxyImage:    testImageName6,
			ApacheExporterImage:        testImageName7,
		},
	}
}

func checkWMSLabels(labelSets ...map[string]string) {
	expectedLabels := map[string]string{
		"app":                          "mapserver",
		"dataset":                      "dataset",
		"dataset-owner":                "owner",
		"service-type":                 "wms",
		"service-version":              "1.0.0",
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/name":       "mapserver-operator",
		"inspire":                      "true",
	}
	for _, labelSet := range labelSets {
		Expect(labelSet).To(Equal(expectedLabels))
	}
}

func getInitContainer(name string, deployment *appsv1.Deployment) (corev1.Container, error) {
	for _, container := range deployment.Spec.Template.Spec.InitContainers {
		if container.Name == name {
			return container, nil
		}
	}

	return corev1.Container{}, fmt.Errorf("init container with name %s not found", name)
}
