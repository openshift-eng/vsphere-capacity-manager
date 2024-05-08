package controller

import (
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"sync"
)

var (
	poolsMu  sync.Mutex
	pools    = make(map[string]*v1.Pool)
	leases   = make(map[string]*v1.Lease)
	networks = make(map[string]*v1.Network)
)
