package main

import (
	"log"
	"os"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/controller"
	"k8s.io/klog/v2/textlogger"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	logger := textlogger.NewLogger(textlogger.NewConfig())
	ctrl.SetLogger(logger)

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		log.Printf("could not create manager: %v", err)
		os.Exit(1)
	}

	v1.AddToScheme(mgr.GetScheme())

	if err := (&controller.LeaseReconciler{}).
		SetupWithManager(mgr); err != nil {
		log.Printf("unable to create controller: %v", err)
		os.Exit(1)
	}

	if err := (&controller.PoolReconciler{}).
		SetupWithManager(mgr); err != nil {
		log.Printf("unable to create controller: %v", err)
		os.Exit(1)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Printf("could not start manager: %v", err)
		os.Exit(1)
	}

}
