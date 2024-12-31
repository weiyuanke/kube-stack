/*
Copyright 2022.

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
	"flag"
	"net/http"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	//"github.com/syndtr/goleveldb/leveldb"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	centralprobev1 "kube-stack.me/apis/centralprobe/v1"
	podlimiterv1 "kube-stack.me/apis/podlimiter/v1"
	podmarkerv1 "kube-stack.me/apis/podmarker/v1"
	slov1beta1 "kube-stack.me/apis/slo/v1beta1"
	centralprobecontrollers "kube-stack.me/controllers/centralprobe"
	podlimitercontrollers "kube-stack.me/controllers/podlimiter"
	podmarkercontrollers "kube-stack.me/controllers/podmarker"
	slocontrollers "kube-stack.me/controllers/slo"
	"kube-stack.me/pkg/debugapi"
	podwebhook "kube-stack.me/webhooks/pods"
	//+kubebuilder:scaffold:imports
)

var (
	scheme           = runtime.NewScheme()
	setupLog         = ctrl.Log.WithName("setup")
	cacheSyncTimeout = 600 * time.Second
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(podmarkerv1.AddToScheme(scheme))
	utilruntime.Must(centralprobev1.AddToScheme(scheme))
	utilruntime.Must(podlimiterv1.AddToScheme(scheme))
	utilruntime.Must(slov1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var leaderElectionNamespace string
	var webhookCertDir string
	var probeAddr string
	var staticFileDirector string
	var dbPath string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionNamespace, "leader-election-namespace", "kube-system", "leader election namespace")
	flag.StringVar(&webhookCertDir, "webhook-cert-directory", ".", "webhook cert directory: tls.crt/tls.key")
	flag.StringVar(&staticFileDirector, "static-file-dir", ".", "root directory for static files")
	flag.StringVar(&dbPath, "db-path", "leveldb_data", "path for leveldb")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	config := ctrl.GetConfigOrDie()
	config.QPS = 500
	config.Burst = 500
	config.UserAgent = "kube-stack"
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"

	dynamicClient := dynamic.NewForConfigOrDie(config)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		setupLog.Error(err, "unable to gen clientset")
		os.Exit(1)
	}

	// initialize leveldb
	// dataBase, err := leveldb.OpenFile(dbPath, nil)
	// if err != nil {
	// 	setupLog.Error(err, "unable to open leveldb")
	// 	os.Exit(1)
	// }

	// defer dataBase.Close()

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      metricsAddr,
		Port:                    9443,
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionNamespace: leaderElectionNamespace,
		CertDir:                 webhookCertDir,
		LeaderElectionID:        "4b2f493f.kube-stack.me",
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
		Controller: v1alpha1.ControllerConfigurationSpec{
			CacheSyncTimeout: &cacheSyncTimeout,
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&podmarkercontrollers.Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PodMarker")
		os.Exit(1)
	}
	if err = (&centralprobecontrollers.Reconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("CentralProbeReconciler"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CentralProbe")
		os.Exit(1)
	}
	if err = (&podlimitercontrollers.PodlimiterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Podlimiter")
		os.Exit(1)
	}
	if err = (&slocontrollers.ResourceStateTransitionReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		DynamicClient: dynamicClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ResourceStateTransition")
		os.Exit(1)
	}
	if err = (&slocontrollers.WatchSLOReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		DynamicClient: dynamicClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "WatchSLO")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Setup webhooks
	setupLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	setupLog.Info("registering webhooks to the webhook server")
	hookServer.Register(
		"/mutating-pod",
		&webhook.Admission{
			Handler: &podwebhook.PodMutate{Client: mgr.GetClient(), ClientSet: clientset},
		},
	)
	hookServer.Register(
		"/validating-pod",
		&webhook.Admission{
			Handler: &podwebhook.PodValidate{Client: mgr.GetClient(), ClientSet: clientset},
		},
	)

	// registe static file server
	hookServer.Register("/", http.FileServer(http.Dir(staticFileDirector)))

	// registe debug api
	debugapi.RegisteToServer(hookServer)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
