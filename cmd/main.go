package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	docsv1 "github.com/BombartSimon/redokube/api/v1"
	"github.com/BombartSimon/redokube/pkg/controller"
	"github.com/BombartSimon/redokube/pkg/redoc"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	// Utiliser une variable pour récupérer le chemin du kubeconfig
	kubeconfigPath string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(docsv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var port int
	var externalURL string
	var specDirectory string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	// Le flag kubeconfig est déjà défini par controller-runtime, on utilise la variable pour le récupérer
	kubeconfigPath = flag.Lookup("kubeconfig").Value.String()
	flag.IntVar(&port, "port", 8080, "The port for the documentation server.")
	flag.StringVar(&externalURL, "external-url", "", "The external URL for the documentation server.")
	flag.StringVar(&specDirectory, "spec-directory", "/tmp/redokube-specs", "The directory to store OpenAPI specs.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Get Kubernetes config
	var config *rest.Config
	var err error
	if kubeconfigPath == "" {
		setupLog.Info("Using in-cluster configuration")
		config, err = rest.InClusterConfig()
	} else {
		setupLog.Info("Using kubeconfig", "path", kubeconfigPath)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	if err != nil {
		setupLog.Error(err, "Failed to get Kubernetes config")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "redokube-leader-election",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create and configure the Redoc server
	server := redoc.NewServer(
		redoc.WithPort(port),
		redoc.WithExternalURL(externalURL),
		redoc.WithSpecDirectory(specDirectory),
	)

	// Start the server in a separate goroutine
	go func() {
		setupLog.Info("Starting Redoc server", "port", port)
		if err := server.Start(); err != nil {
			setupLog.Error(err, "Failed to start Redoc server")
			os.Exit(1)
		}
	}()

	// Create the controller
	if err = (&controller.OpenAPISpecReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Server: server,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpenAPISpec")
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

	// Handle graceful shutdown
	setupLog.Info("setting up signal handler")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		setupLog.Info("Shutting down Redoc server...")
		if err := server.Stop(); err != nil {
			setupLog.Error(err, "Error shutting down Redoc server")
		}
	}()

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
