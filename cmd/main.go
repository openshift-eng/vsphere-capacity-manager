package main

import (
	"log"
	"os"

	"k8s.io/klog/v2/textlogger"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	v1 "github.com/openshift-eng/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-eng/vsphere-capacity-manager/pkg/controller"
)

func main() {
	logger := textlogger.NewLogger(textlogger.NewConfig())
	ctrl.SetLogger(logger)

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		log.Printf("could not create manager: %v", err)
		os.Exit(1)
	}

	err = v1.AddToScheme(mgr.GetScheme())
	if err != nil {
		log.Printf("could not add types to scheme: %v", err)
		os.Exit(1)
	}

	controller.InitMetrics()

	if err := (&controller.PoolReconciler{}).
		SetupWithManager(mgr); err != nil {
		log.Printf("unable to create controller: %v", err)
		os.Exit(1)
	}

	if err := (&controller.LeaseReconciler{
		// This will be set for now via constant, but might be good in future to make configurable via startup parameter.
		AllowMultiToUseSingle: controller.ALLOW_MULTI_TO_USE_SINGLE,
	}).
		SetupWithManager(mgr); err != nil {
		log.Printf("unable to create controller: %v", err)
		os.Exit(1)
	}

	if err := (&controller.NetworkReconciler{}).
		SetupWithManager(mgr); err != nil {
		log.Printf("unable to create controller: %v", err)
		os.Exit(1)
	}

	if err := (&controller.NamespaceReconciler{}).
		SetupWithManager(mgr); err != nil {
		log.Printf("unable to create controller: %v", err)
		os.Exit(1)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Printf("could not start manager: %v", err)
		os.Exit(1)
	}

}
