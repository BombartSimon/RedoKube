package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	docsv1 "github.com/BombartSimon/redokube/api/v1"
	"github.com/BombartSimon/redokube/pkg/redoc"
)

// OpenAPISpecReconciler reconciles a OpenAPISpec object
type OpenAPISpecReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Server *redoc.Server
}

// +kubebuilder:rbac:groups=docs.redokube.io,resources=openapispecs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=docs.redokube.io,resources=openapispecs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=docs.redokube.io,resources=openapispecs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *OpenAPISpecReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling OpenAPISpec", "namespacedName", req.NamespacedName)

	// Fetch the OpenAPISpec instance
	openAPISpec := &docsv1.OpenAPISpec{}
	err := r.Get(ctx, req.NamespacedName, openAPISpec)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request
			logger.Info("OpenAPISpec resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		logger.Error(err, "Failed to get OpenAPISpec")
		return ctrl.Result{}, err
	}

	// Initialize status if it's a new resource
	if openAPISpec.Status.Status == "" {
		openAPISpec.Status.Status = "Pending"
		if err := r.Status().Update(ctx, openAPISpec); err != nil {
			logger.Error(err, "Failed to update OpenAPISpec status")
			return ctrl.Result{}, err
		}
	}

	// Process the OpenAPISpec
	specURL, err := r.Server.RegisterSpec(openAPISpec)
	if err != nil {
		openAPISpec.Status.Status = "Failed"
		openAPISpec.Status.ErrorMessage = err.Error()
		if updateErr := r.Status().Update(ctx, openAPISpec); updateErr != nil {
			logger.Error(updateErr, "Failed to update OpenAPISpec status after error")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	// Update status on success
	openAPISpec.Status.Status = "Available"
	openAPISpec.Status.URL = specURL
	openAPISpec.Status.LastUpdated.Time = time.Now()
	openAPISpec.Status.ErrorMessage = ""

	if err := r.Status().Update(ctx, openAPISpec); err != nil {
		logger.Error(err, "Failed to update OpenAPISpec status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *OpenAPISpecReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&docsv1.OpenAPISpec{}).
		Complete(r)
}
