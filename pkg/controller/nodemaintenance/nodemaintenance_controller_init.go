// +build !test

package nodemaintenance

import (
	"k8s.io/client-go/kubernetes"
	nodemaintenanceapi "kubevirt.io/node-maintenance-operator/pkg/apis/nodemaintenance/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new NodeMaintenance Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	r, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, r)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (*ReconcileNodeMaintenance, error) {

	cs, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	r := &ReconcileNodeMaintenance{
		client:          mgr.GetClient(),
		scheme:          mgr.GetScheme(),
		clientset:       cs,
		leaseCallbackCh: make(chan event.GenericEvent, 1),
	}

	err = initDrainer(r, cs)
	return r, err
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileNodeMaintenance) error {
	// Create a new controller
	c, err := controller.New("nodemaintenance-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			newObj := e.ObjectNew.(*nodemaintenanceapi.NodeMaintenance)
			return !newObj.DeletionTimestamp.IsZero()
		},
	}

	// Create a source for watching noe maintenance events.
	src := &source.Kind{Type: &nodemaintenanceapi.NodeMaintenance{}}

	// Watch for changes to primary resource NodeMaintenance
	err = c.Watch(src, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	// Watch for events from the lease handler
	leaseCallbackSrc := &source.Channel{Source: r.leaseCallbackCh}
	err = c.Watch(leaseCallbackSrc, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}
