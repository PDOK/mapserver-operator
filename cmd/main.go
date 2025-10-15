/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"os"

	"github.com/pdok/mapserver-operator/internal/controller/types"

	"github.com/go-logr/zapr"
	"github.com/pdok/smooth-operator/pkg/integrations/logging"
	"github.com/peterbourgon/ff"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/pdok/mapserver-operator/internal/controller/mapfilegenerator"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	pdoknlv3 "github.com/pdok/mapserver-operator/api/v3"
	"github.com/pdok/mapserver-operator/internal/controller"
	webhookpdoknlv3 "github.com/pdok/mapserver-operator/internal/webhook/v3"
	// +kubebuilder:scaffold:imports
)

const (
	EnvFalse = "false"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(traefikiov1alpha1.AddToScheme(scheme))
	utilruntime.Must(smoothoperatorv1.AddToScheme(scheme))
	utilruntime.Must(pdoknlv3.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

//nolint:funlen
func main() {
	var metricsAddr string
	var certDir string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	var host string
	var mapserverDebugLevel int
	var multitoolImage, mapfileGeneratorImage, mapserverImage, capabilitiesGeneratorImage, featureinfoGeneratorImage, ogcWebserviceProxyImage, apacheExporterImage string
	var slackWebhookURL string
	var logLevel int
	var setUptimeOperatorAnnotations bool
	var storageClassName string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&certDir, "cert-dir", "", "CertDir contains the webhook server key and certificate. Defaults to <temp-dir>/k8s-webhook-server/serving-certs.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.StringVar(&host, "baseurl", "", "The host which is used in the mapserver service.")
	flag.StringVar(&multitoolImage, "multitool-image", "", "The image to use in the blob download init-container.")
	flag.StringVar(&mapfileGeneratorImage, "mapfile-generator-image", "", "The image to use in the mapfile generator init-container.")
	flag.StringVar(&mapserverImage, "mapserver-image", "", "The image to use in the mapserver container.")
	flag.StringVar(&capabilitiesGeneratorImage, "capabilities-generator-image", "", "The image to use in the capabilities generator init-container.")
	flag.StringVar(&featureinfoGeneratorImage, "featureinfo-generator-image", "", "The image to use in the featureinfo generator init-container.")
	flag.StringVar(&ogcWebserviceProxyImage, "ogc-webservice-proxy-image", "", "The image to use in the ogc webservice proxy container.")
	flag.StringVar(&apacheExporterImage, "apache-exporter-image", "", "The image to use in the apache-exporter container.")
	flag.IntVar(&mapserverDebugLevel, "mapserver-debug-level", 0, "Debug level for the mapserver container, between 0 (error only) and 5 (very very verbose).")
	flag.StringVar(&slackWebhookURL, "slack-webhook-url", "", "The webhook url for sending slack messages. Disabled if left empty")
	flag.IntVar(&logLevel, "log-level", 0, "The zapcore loglevel. 0 = info, 1 = warn, 2 = error")
	flag.BoolVar(&setUptimeOperatorAnnotations, "set-uptime-operator-annotations", true, "When enabled IngressRoutes get annotations that are used by the pdok/uptime-operator.")
	flag.StringVar(&storageClassName, "storage-class-name", "", "The name of the storage class to use when using an ephemeral volume.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)

	if err := ff.Parse(flag.CommandLine, os.Args[1:], ff.WithEnvVarNoPrefix()); err != nil {
		setupLog.Error(err, "unable to parse flags")
		os.Exit(1)
	}

	//nolint:gosec
	levelEnabler := zapcore.Level(logLevel)
	zapLogger, _ := logging.SetupLogger("mapserver-operator", slackWebhookURL, levelEnabler)
	logrLogger := zapr.NewLogger(zapLogger)
	ctrl.SetLogger(logrLogger)

	reqFlags := make(map[string]string)
	reqFlags["baseurl"] = host
	reqFlags["multitool-image"] = multitoolImage
	reqFlags["mapfile-generator-image"] = mapfileGeneratorImage
	reqFlags["mapserver-image"] = mapserverImage
	reqFlags["capabilities-generator-image"] = capabilitiesGeneratorImage
	reqFlags["featureinfo-generator-image"] = featureinfoGeneratorImage
	reqFlags["ogc-webservice-proxy-image"] = ogcWebserviceProxyImage
	reqFlags["apache-exporter-image"] = apacheExporterImage

	for reqFlag, val := range reqFlags {
		if val == "" {
			setupLog.Error(errors.New(reqFlag+" is a required flag"), "A value for "+reqFlag+" must be specified.")
			os.Exit(1)
		}
	}

	pdoknlv3.SetHost(host)
	mapfilegenerator.SetDebugLevel(mapserverDebugLevel)
	controller.SetUptimeOperatorAnnotations(setUptimeOperatorAnnotations)
	controller.SetStorageClassName(storageClassName)

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		CertDir: certDir,
		TLSOpts: tlsOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "01a58011.pdok.nl",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.WMSReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Images: types.Images{
			MultitoolImage:             multitoolImage,
			MapfileGeneratorImage:      mapfileGeneratorImage,
			MapserverImage:             mapserverImage,
			CapabilitiesGeneratorImage: capabilitiesGeneratorImage,
			FeatureinfoGeneratorImage:  featureinfoGeneratorImage,
			OgcWebserviceProxyImage:    ogcWebserviceProxyImage,
			ApacheExporterImage:        apacheExporterImage,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "WMS")
		os.Exit(1)
	}
	if err = (&controller.WFSReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Images: types.Images{
			MultitoolImage:             multitoolImage,
			MapfileGeneratorImage:      mapfileGeneratorImage,
			MapserverImage:             mapserverImage,
			CapabilitiesGeneratorImage: capabilitiesGeneratorImage,
			ApacheExporterImage:        apacheExporterImage,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "WFS")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != EnvFalse {
		if err = webhookpdoknlv3.SetupWFSWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "WFS")
			os.Exit(1)
		}
	}

	if os.Getenv("ENABLE_WEBHOOKS") != EnvFalse {
		if err = webhookpdoknlv3.SetupWMSWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "WMS")
			os.Exit(1)
		}
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
