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

	types2 "github.com/pdok/mapserver-operator/internal/controller/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pdok/mapserver-operator/internal/controller/utils"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorsamples "github.com/pdok/smooth-operator/config/samples"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	smoothoperatorvalidation "github.com/pdok/smooth-operator/pkg/validation"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	wfsResourceName = "wfs-resource"
)

var _ = Describe("WFS Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		// Setup variables for unique WFS resource per It node
		counter := 1
		var typeNamespacedNameWfs types.NamespacedName

		wfs := &pdoknlv3.WFS{}

		typeNamespacedNameOwnerInfo := types.NamespacedName{
			Namespace: namespace,
			Name:      ownerInfoResourceName,
		}
		ownerInfo := &smoothoperatorv1.OwnerInfo{}
		initScripts, includeMapfileGeneratorConfigMap := true, true

		BeforeEach(func() {
			pdoknlv3.SetHost("localhost")

			// Create a unique WFS resource for every It node to prevent unexpected resource state caused by finalizers
			sampleWfs, err := getUniqueWFSSample(counter)
			Expect(err).To(BeNil())
			typeNamespacedNameWfs = getUniqueWfsTypeNamespacedName(counter)
			counter++

			// Set most used options
			sampleWfs.Spec.Options.PrefetchData = true

			By("creating the custom resource for the Kind WFS")
			err = k8sClient.Get(ctx, typeNamespacedNameWfs, wfs)
			if err != nil && apierrors.IsNotFound(err) {
				resource := sampleWfs.DeepCopy()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedNameWfs, wfs)).To(Succeed())
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
			wfsResource := &pdoknlv3.WFS{}
			wfsResource.Namespace = namespace
			wfsResource.Name = typeNamespacedNameWfs.Name
			err := k8sClient.Get(ctx, typeNamespacedNameWfs, wfsResource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WFS")
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, wfsResource))).To(Succeed())

			ownerInfoResource := &smoothoperatorv1.OwnerInfo{}
			err = k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfoResource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance OwnerInfo")
			Expect(k8sClient.Delete(ctx, ownerInfoResource)).To(Succeed())

			// the testEnv does not do garbage collection (https://book.kubebuilder.io/reference/envtest#testing-considerations)
			By("Cleaning Owned Resources")
			objects, err := getExpectedObjects(ctx, wfs, initScripts, includeMapfileGeneratorConfigMap)
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
			controllerReconciler := getWFSReconciler()

			By("Reconciling the created resource")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			By("Waiting for the owned resources to be created")
			initScripts = false
			expectedBareObjects, err := getExpectedObjects(ctx, wfs, initScripts, includeMapfileGeneratorConfigMap)
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

			By("Checking the status of the WFS")
			err = k8sClient.Get(ctx, typeNamespacedNameWfs, wfs)
			Expect(err).NotTo(HaveOccurred())
			// TODO fix
			Expect(len(wfs.Status.Conditions)).To(BeEquivalentTo(1))
			Expect(wfs.Status.Conditions[0].Status).To(BeEquivalentTo(metav1.ConditionTrue))
		})

		It("Should successfully reconcile after a change in an owned resource", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS, checking the finalizer, and reconciling again")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			By("Getting the original Deployment")
			deployment := getBareDeployment(wfs)
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

			By("Reconciling the WFS again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was restored")
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(*deployment.Spec.RevisionHistoryLimit).To(BeEquivalentTo(originalRevisionHistoryLimit))
			}, "10s", "1s").Should(BeTrue())
		})

		It("Should create correct deployment manifest.", func() {
			controllerReconciler := getWFSReconciler()
			reconcilerImages := getReconcilerImages(controllerReconciler)

			By("Reconciling the WFS and checking the deployment")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			deployment := getBareDeployment(wfs)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(deployment.GetName()).Should(Equal(wfs.GetName() + "-wfs-mapserver"))
			Expect(deployment.GetNamespace()).Should(Equal(namespace))

			Expect(deployment.Spec.Template.Spec.TerminationGracePeriodSeconds).Should(Equal(smoothoperatorutils.Pointer(int64(60))))

			/**
			Label + selector tests
			*/
			checkWFSLabels(deployment.GetLabels(), deployment.Spec.Selector.MatchLabels)

			/**
			Container tests
			*/
			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.Name).Should(Equal(utils.MapserverName))
			Expect(container.Ports[0].ContainerPort).Should(Equal(int32(80)))
			Expect(container.Image).Should(Equal(reconcilerImages.MapserverImage))
			Expect(container.ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
			Expect(container.Resources.Limits.Memory().String()).Should(Equal("12M"))
			Expect(container.Resources.Requests.Cpu().String()).Should(Equal("150m"))
			Expect(len(container.LivenessProbe.Exec.Command)).Should(Equal(3))
			Expect(container.LivenessProbe.Exec.Command[2]).Should(Equal("wget -SO- -T 10 -t 2 'http://127.0.0.1:80/mapserver?SERVICE=WFS&request=GetCapabilities' 2>&1 | egrep -aiA10 'HTTP/1.1 200' | egrep -i 'Content-Type: text/xml'"))
			Expect(container.LivenessProbe.FailureThreshold).Should(Equal(int32(3)))
			Expect(container.LivenessProbe.InitialDelaySeconds).Should(Equal(int32(20)))
			Expect(container.LivenessProbe.PeriodSeconds).Should(Equal(int32(10)))
			Expect(container.LivenessProbe.TimeoutSeconds).Should(Equal(int32(10)))
			Expect(len(container.ReadinessProbe.Exec.Command)).Should(Equal(3))
			Expect(container.ReadinessProbe.Exec.Command[2]).Should(Equal("wget -SO- -T 10 -t 2 'http://127.0.0.1:80/mapserver?SERVICE=WFS&VERSION=2.0.0&REQUEST=GetFeature&TYPENAMES=FeatureType1&STARTINDEX=0&COUNT=1' 2>&1 | egrep -aiA10 'HTTP/1.1 200' | egrep -i 'Content-Type: text/xml'"))
			Expect(container.ReadinessProbe.FailureThreshold).Should(Equal(int32(3)))
			Expect(container.ReadinessProbe.InitialDelaySeconds).Should(Equal(int32(20)))
			Expect(container.ReadinessProbe.PeriodSeconds).Should(Equal(int32(10)))
			Expect(container.ReadinessProbe.TimeoutSeconds).Should(Equal(int32(10)))
			Expect(len(container.StartupProbe.Exec.Command)).Should(Equal(3))
			Expect(container.StartupProbe.Exec.Command[2]).Should(Equal("wget -SO- -T 10 -t 2 'http://127.0.0.1:80/mapserver?SERVICE=WFS&VERSION=2.0.0&REQUEST=GetFeature&TYPENAMES=FeatureType1,FeatureType2&STARTINDEX=0&COUNT=1' 2>&1 | egrep -aiA10 'HTTP/1.1 200' | egrep -i 'Content-Type: text/xml'"))
			Expect(container.StartupProbe.FailureThreshold).Should(Equal(int32(3)))
			Expect(container.StartupProbe.InitialDelaySeconds).Should(Equal(int32(20)))
			Expect(container.StartupProbe.PeriodSeconds).Should(Equal(int32(10)))
			Expect(container.StartupProbe.TimeoutSeconds).Should(Equal(int32(10)))

			/**
			Init container tests
			*/
			getInitContainer := func(name string) (corev1.Container, error) {
				for _, container := range deployment.Spec.Template.Spec.InitContainers {
					if container.Name == name {
						return container, nil
					}
				}

				return corev1.Container{}, fmt.Errorf("init container with name %s not found", name)
			}

			blobDownloadContainer, err := getInitContainer(utils.BlobDownloadName)
			Expect(err).NotTo(HaveOccurred())
			Expect(blobDownloadContainer.Image).Should(Equal(reconcilerImages.MultitoolImage))
			volumeMounts := []corev1.VolumeMount{
				{Name: "base", MountPath: "/srv/data"},
				{Name: "data", MountPath: "/var/www"},
				{Name: utils.InitScriptsName, MountPath: "/srv/scripts", ReadOnly: true},
			}
			envFrom := []corev1.EnvFromSource{
				utils.NewEnvFromSource(utils.EnvFromSourceTypeConfigMap, "blobs-testtest"),
				utils.NewEnvFromSource(utils.EnvFromSourceTypeSecret, "blobs-testtest"),
			}
			Expect(blobDownloadContainer.VolumeMounts).Should(Equal(volumeMounts))
			Expect(blobDownloadContainer.EnvFrom).Should(Equal(envFrom))
			Expect(blobDownloadContainer.Command).Should(Equal([]string{"/bin/sh", "-c"}))
			Expect(len(blobDownloadContainer.Args)).Should(BeNumerically(">", 0))

			mapfileGeneratorContainer, err := getInitContainer(utils.MapfileGeneratorName)
			Expect(err).NotTo(HaveOccurred())
			Expect(mapfileGeneratorContainer.Image).Should(Equal(reconcilerImages.MapfileGeneratorImage))
			volumeMounts = []corev1.VolumeMount{
				{Name: "base", MountPath: "/srv/data"},
				utils.GetConfigVolumeMount(utils.ConfigMapMapfileGeneratorVolumeName),
			}
			Expect(mapfileGeneratorContainer.VolumeMounts).Should(Equal(volumeMounts))
			Expect(mapfileGeneratorContainer.Command).Should(Equal([]string{"generate-mapfile"}))
			Expect(mapfileGeneratorContainer.Args).Should(Equal([]string{"--not-include", "wfs", "/input/input.json", "/srv/data/config/mapfile"}))

			capabilitiesGeneratorContainer, err := getInitContainer(utils.CapabilitiesGeneratorName)
			Expect(err).NotTo(HaveOccurred())
			Expect(capabilitiesGeneratorContainer.Image).Should(Equal(reconcilerImages.CapabilitiesGeneratorImage))
			volumeMounts = []corev1.VolumeMount{
				{Name: "data", MountPath: "/var/www"},
				utils.GetConfigVolumeMount(utils.ConfigMapCapabilitiesGeneratorVolumeName),
			}
			Expect(capabilitiesGeneratorContainer.VolumeMounts).Should(Equal(volumeMounts))
			env := []corev1.EnvVar{
				{Name: "SERVICECONFIG", Value: "/input/input.yaml"},
			}
			Expect(capabilitiesGeneratorContainer.Env).Should(Equal(env))

			/**
			Volumes tests
			*/
			expectedVolumes := []string{"" +
				"base",
				"data",
				utils.MapserverName,
				utils.InitScriptsName,
				utils.ConfigMapCapabilitiesGeneratorVolumeName,
				utils.ConfigMapMapfileGeneratorVolumeName,
			}
			for _, ev := range expectedVolumes {
				Expect(slices.IndexFunc(deployment.Spec.Template.Spec.Volumes, func(v corev1.Volume) bool {
					return v.Name == ev
				})).ShouldNot(BeEquivalentTo(-1))
			}

		})

		It("Should not mount a blob download configmap if options.prefetchData is false.", func() {
			wfsResource := &pdoknlv3.WFS{}
			wfsResource.Namespace = namespace
			wfsResource.Name = typeNamespacedNameWfs.Name
			err := k8sClient.Get(ctx, typeNamespacedNameWfs, wfsResource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WFS")
			Expect(k8sClient.Delete(ctx, wfsResource)).To(Succeed())

			sampleWfs, err := getUniqueWFSSample(9999)
			typeNamespacedNameWfs.Name = sampleWfs.Name
			Expect(err).NotTo(HaveOccurred())
			sampleWfs.Spec.Options.PrefetchData = false
			Expect(k8sClient.Create(ctx, sampleWfs.DeepCopy())).To(Succeed())
			Expect(k8sClient.Get(ctx, typeNamespacedNameWfs, wfs)).To(Succeed())

			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the configMap")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			_, err = getHashedConfigMapNameFromClient(ctx, wfs, utils.InitScriptsName)
			Expect(err).To(HaveOccurred())
		})

		It("Should create correct configMap manifest.", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the configMap")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			configMap := getBareConfigMap(wfs, utils.MapserverName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wfs, utils.MapserverName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wfs.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			// Make sure the name is hashed
			Expect(configMap.GetName()).To(HavePrefix(wfs.GetName() + "-wfs-mapserver"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))

			checkWFSLabels(configMap.GetLabels())

			defaultMapserverConf, ok := configMap.Data["default_mapserver.conf"]
			Expect(ok).To(BeTrue())
			Expect(defaultMapserverConf).To(ContainSubstring("MAP \"/srv/data/config/mapfile/service.map\""))

			includeConf, ok := configMap.Data["include.conf"]
			Expect(ok).To(BeTrue())
			Expect(includeConf).To(ContainSubstring("/eigenaar/dataset/wfs"))

			ogcLua, ok := configMap.Data["ogc.lua"]
			Expect(ok).To(BeTrue())
			Expect(ogcLua).To(ContainSubstring("/srv/mapserver/config/scraping-error.xml"))

			scrapingErrorXML, ok := configMap.Data["scraping-error.xml"]
			Expect(ok).To(BeTrue())
			Expect(scrapingErrorXML).To(ContainSubstring("It is not possible to use a 'startindex' higher than 50.000"))
		})

		It("Should create correct configMapMapfileGenerator manifest.", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the configMap")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			configMap := getBareConfigMap(wfs, utils.MapfileGeneratorName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wfs, utils.ConfigMapMapfileGeneratorVolumeName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wfs.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(configMap.GetName()).To(HavePrefix(wfs.GetName() + "-wfs-mapfile-generator-"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))
			checkWFSLabels(configMap.GetLabels())

			data, ok := configMap.Data["input.json"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))
			// input.json content is tested in mapfilegenerator/mapfile_generator_test.go
		})

		It("Should create correct configMapBlobDownload manifest.", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the configMap")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			configMap := getBareConfigMap(wfs, utils.InitScriptsName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wfs, utils.InitScriptsName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wfs.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(configMap.GetName()).To(HavePrefix(wfs.GetName() + "-wfs-init-scripts-"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))
			checkWFSLabels(configMap.GetLabels())

			data, ok := configMap.Data["gpkg_download.sh"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))
			// gpkg_download.sh content is tested in blobdownload/blob_download_test.go
		})

		It("Should create correct configMapCapabilitiesGenerator manifest.", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the configMap")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			configMap := getBareConfigMap(wfs, utils.CapabilitiesGeneratorName)
			configMapName, err := getHashedConfigMapNameFromClient(ctx, wfs, utils.ConfigMapCapabilitiesGeneratorVolumeName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: wfs.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(configMap.GetName()).To(HavePrefix(wfs.GetName() + "-wfs-capabilities-generator-"))
			Expect(configMap.GetNamespace()).To(Equal(namespace))
			Expect(configMap.Immutable).To(Equal(smoothoperatorutils.Pointer(true)))
			checkWFSLabels(configMap.GetLabels())

			data, ok := configMap.Data["input.yaml"]
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))
			// input.yaml content is tested in capabilitiesgenerator/capabilities_generator_test.go
		})

		It("Should create correct middlewareCorsHeaders manifest.", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the middlewareCorsHeaders")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			middlewareCorsHeaders := getBareCorsHeadersMiddleware(wfs)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(middlewareCorsHeaders), middlewareCorsHeaders)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(middlewareCorsHeaders.Name).Should(Equal(wfs.GetName() + "-wfs-mapserver-headers"))
			Expect(middlewareCorsHeaders.Namespace).Should(Equal("default"))
			checkWFSLabels(middlewareCorsHeaders.GetLabels())
			// Expect(middlewareCorsHeaders.Spec.Headers.FrameDeny).Should(Equal(true))
			Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Headers"]).Should(Equal("Content-Type"))
			Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Method"]).Should(Equal("GET, POST, OPTIONS"))
			Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Origin"]).Should(Equal("*"))
		})

		It("Should create correct podDisruptionBudget manifest.", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the podDisruptionBudget")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			podDisruptionBudget := getBarePodDisruptionBudget(wfs)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			/**
			Label + selector tests
			*/
			checkWFSLabels(podDisruptionBudget.GetLabels(), podDisruptionBudget.Spec.Selector.MatchLabels)

			Expect(podDisruptionBudget.GetName()).To(Equal(wfs.GetName() + "-wfs-mapserver"))
			Expect(podDisruptionBudget.Spec.MaxUnavailable.IntValue()).Should(Equal(1))
		})

		It("Should create correct horizontalPodAutoScaler manifest.", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the horizontalPodAutoScaler")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			autoscaler := getBareHorizontalPodAutoScaler(wfs)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(autoscaler), autoscaler)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(autoscaler.GetName()).To(Equal(wfs.GetName() + "-wfs-mapserver"))
			Expect(autoscaler.Spec.ScaleTargetRef).To(Equal(autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       wfs.GetName() + "-wfs-mapserver",
			}))

			/**
			Label + selector tests
			*/
			checkWFSLabels(autoscaler.GetLabels())

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
				StabilizationWindowSeconds: smoothoperatorutils.Pointer(int32(300)),
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
							AverageUtilization: smoothoperatorutils.Pointer(int32(60)),
						},
					},
				},
			}))
		})

		It("Should create correct service manifest.", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the service")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			service := getBareService(wfs)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(service), service)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(service.GetName()).To(Equal(wfs.GetName() + "-wfs-mapserver"))
			Expect(service.Spec.Ports).To(Equal([]corev1.ServicePort{
				{
					Name:       utils.MapserverName,
					Port:       80,
					TargetPort: intstr.FromInt32(80),
					Protocol:   corev1.ProtocolTCP,
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
			checkWFSLabels(service.GetLabels(), service.Spec.Selector)
		})

		It("Should create correct ingressRoute manifest.", func() {
			controllerReconciler := getWFSReconciler()

			By("Reconciling the WFS and checking the ingressRoute")
			reconcileWFS(controllerReconciler, wfs, typeNamespacedNameWfs)

			ingressRoute := getBareIngressRoute(wfs)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ingressRoute), ingressRoute)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			checkWFSLabels(ingressRoute.GetLabels())

			Expect(ingressRoute.Annotations).To(Equal(map[string]string{
				"uptime.pdok.nl/id":   utils.Sha1Hash(wfs.TypedName()),
				"uptime.pdok.nl/name": "WFS resource 13 INSPIRE WFS",
				"uptime.pdok.nl/tags": "public-stats,wfs,inspire",
				"uptime.pdok.nl/url":  "https://service.pdok.nl/eigenaar/dataset/wfs/1.0.0?SERVICE=WFS&VERSION=2.0.0&REQUEST=GetFeature&TYPENAMES=FeatureType1&STARTINDEX=0&COUNT=1",
			}))

			Expect(ingressRoute.GetName()).To(Equal(wfs.GetName() + "-wfs-mapserver"))
			Expect(len(ingressRoute.Spec.Routes)).To(Equal(1))
			Expect(ingressRoute.Spec.Routes[0]).To(Equal(traefikiov1alpha1.Route{
				Kind:        "Rule",
				Match:       "Host(`localhost`) && Path(`/eigenaar/dataset/wfs/1.0.0`)",
				Middlewares: []traefikiov1alpha1.MiddlewareRef{{Name: wfs.GetName() + "-wfs-mapserver-headers"}},
				Services: []traefikiov1alpha1.Service{{
					LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
						Kind: "Service",
						Port: intstr.FromInt32(80),
						Name: wfs.GetName() + "-wfs-mapserver",
					},
				}},
			}))
		})
	})
})

func reconcileWFS(r *WFSReconciler, wfs *pdoknlv3.WFS, typeNamespacedNameWfs types.NamespacedName) {
	// Reconcile the WFS
	_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameWfs})
	Expect(err).NotTo(HaveOccurred())

	// Check it's there
	err = k8sClient.Get(ctx, typeNamespacedNameWfs, wfs)
	Expect(err).NotTo(HaveOccurred())
}

//go:embed test_manifests/v3_wfs.yaml
var testManifestWFS []byte

func getUniqueWFSSample(counter int) (*pdoknlv3.WFS, error) {
	sample := &pdoknlv3.WFS{}
	defaulted, err := smoothoperatorvalidation.ApplySchemaDefaultsStr(string(testManifestWFS))
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal([]byte(defaulted), sample)
	if err != nil {
		return nil, err
	}

	sample.Name = getUniqueWfsResourceName(counter)
	sample.Namespace = namespace
	sample.Spec.Service.OwnerInfoRef = ownerInfoResourceName

	return sample, nil
}

func getUniqueWfsTypeNamespacedName(counter int) types.NamespacedName {
	return types.NamespacedName{
		Name:      getUniqueWfsResourceName(counter),
		Namespace: namespace,
	}
}

func getUniqueWfsResourceName(counter int) string {
	return fmt.Sprintf("%s-%v", wfsResourceName, counter)
}

func getWFSReconciler() *WFSReconciler {
	return &WFSReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
		Images: types2.Images{
			MultitoolImage:             testImageName1,
			MapfileGeneratorImage:      testImageName2,
			MapserverImage:             testImageName3,
			CapabilitiesGeneratorImage: testImageName4,
			ApacheExporterImage:        testImageName5,
		},
	}
}

func checkWFSLabels(labelSets ...map[string]string) {
	expectedLabels := map[string]string{
		"app":                          utils.MapserverName,
		"dataset":                      "dataset",
		"dataset-owner":                "eigenaar",
		"service-type":                 "wfs",
		"service-version":              "1.0.0",
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/name":       "mapserver-operator",
		"inspire":                      "true",
	}
	for _, labelSet := range labelSets {
		Expect(labelSet).To(Equal(expectedLabels))
	}
}
