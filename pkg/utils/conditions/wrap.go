package conditions

import (
	v1 "github.com/openshift-eng/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

type LeaseWrapper struct {
	*v1.Lease
}

func (m *LeaseWrapper) GetConditions() []v1.Condition {
	return m.Status.Conditions
}

func (m *LeaseWrapper) SetConditions(conditions []v1.Condition) {
	m.Status.Conditions = conditions
}
