package data

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LeaseSpec struct {
	ResourceSpec
}

type LeaseStatus struct {
	Resource      *Resource `json:"resource"`
	LeasedAt      string    `json:"leased-at"`
	BoskosLeaseID string    `json:"boskos-lease-id"`
	Pool          string    `json:"pool"`
}

type Lease struct {
	Spec              LeaseSpec   `json:"spec"`
	Status            LeaseStatus `json:"status"`
	metav1.ObjectMeta `json:"metadata"`
	metav1.TypeMeta   `json:"type"`
}

type Leases []*Lease
