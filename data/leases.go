package data

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LeaseSpec struct {
	ResourceSpec
	// RequiredPool when configured, this lease can only be
	// scheduled in the required pool.
	RequiredPool string `json:"required-pool"`
}

type LeaseStatus struct {
	LeasedAt      string    `json:"leased-at"`
	BoskosLeaseID string    `json:"boskos-lease-id"`
	Pool          string    `json:"pool"`
	PortGroups    []Network `json:"port-groups"`
}

// Lease defines the resource requirements for a CI job. When fulfilled,
// the lease is assigned to a pool and the resources are assigned to the lease.
type Lease struct {
	Spec              LeaseSpec   `json:"spec"`
	Status            LeaseStatus `json:"status"`
	metav1.ObjectMeta `json:"metadata"`
	metav1.TypeMeta   `json:"type"`
}

type Leases []*Lease
