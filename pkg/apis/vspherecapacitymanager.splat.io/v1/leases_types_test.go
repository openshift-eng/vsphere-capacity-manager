/*
Copyright 2024 Red Hat, Inc.

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

package v1

import (
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestLeaseVCentersValidation tests the validation marker for the VCenters field
// The actual validation happens in the CRD via kubebuilder:validation:Minimum=0
// These tests document expected behavior when creating/updating Lease objects
func TestLeaseVCentersValidation(t *testing.T) {
	tests := []struct {
		name        string
		vcenters    int
		expectValid bool
		description string
	}{
		{
			name:        "zero vcenters is valid (no limit)",
			vcenters:    0,
			expectValid: true,
			description: "When vcenters is 0 or unset, no vcenter limit is applied",
		},
		{
			name:        "positive vcenters is valid",
			vcenters:    1,
			expectValid: true,
			description: "Positive vcenter values are valid",
		},
		{
			name:        "multiple vcenters is valid",
			vcenters:    3,
			expectValid: true,
			description: "Multiple vcenters can be specified",
		},
		{
			name:        "large vcenters value is valid",
			vcenters:    100,
			expectValid: true,
			description: "Large positive values are valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lease := &Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lease",
					Namespace: "default",
				},
				Spec: LeaseSpec{
					VCpus:    16,
					Memory:   32,
					Pools:    2,
					VCenters: tt.vcenters,
				},
			}

			// Marshal to JSON to ensure the field is serializable
			_, err := json.Marshal(lease)
			if err != nil && tt.expectValid {
				t.Errorf("Expected valid lease to marshal successfully, got error: %v", err)
			}

			// Verify the field value is set correctly
			if lease.Spec.VCenters != tt.vcenters {
				t.Errorf("Expected VCenters=%d, got %d", tt.vcenters, lease.Spec.VCenters)
			}
		})
	}
}

// TestLeaseVCentersDefaultValue tests that VCenters defaults to 0 when unset
func TestLeaseVCentersDefaultValue(t *testing.T) {
	lease := &Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: LeaseSpec{
			VCpus:  16,
			Memory: 32,
		},
	}

	// When VCenters is not explicitly set, it should default to 0
	if lease.Spec.VCenters != 0 {
		t.Errorf("Expected default VCenters=0, got %d", lease.Spec.VCenters)
	}
}

// TestLeaseVCentersWithPoolsField tests the interaction between Pools and VCenters
func TestLeaseVCentersWithPoolsField(t *testing.T) {
	tests := []struct {
		name        string
		pools       int
		vcenters    int
		description string
	}{
		{
			name:        "vcenters equals pools",
			pools:       3,
			vcenters:    3,
			description: "Each pool can be on a different vcenter",
		},
		{
			name:        "vcenters less than pools",
			pools:       4,
			vcenters:    2,
			description: "Pools must be distributed across at most 2 vcenters",
		},
		{
			name:        "vcenters greater than pools",
			pools:       2,
			vcenters:    5,
			description: "More vcenters allowed than pools needed (acts as cap)",
		},
		{
			name:        "vcenters zero with multiple pools",
			pools:       5,
			vcenters:    0,
			description: "No vcenter limit - pools can span any number of vcenters",
		},
		{
			name:        "single pool single vcenter",
			pools:       1,
			vcenters:    1,
			description: "Simple case - one pool on one vcenter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lease := &Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lease",
					Namespace: "default",
				},
				Spec: LeaseSpec{
					VCpus:    16,
					Memory:   32,
					Pools:    tt.pools,
					VCenters: tt.vcenters,
				},
			}

			// Verify both fields are set correctly
			if lease.Spec.Pools != tt.pools {
				t.Errorf("Expected Pools=%d, got %d", tt.pools, lease.Spec.Pools)
			}
			if lease.Spec.VCenters != tt.vcenters {
				t.Errorf("Expected VCenters=%d, got %d", tt.vcenters, lease.Spec.VCenters)
			}

			// Document behavior when vcenters is 0
			if tt.vcenters == 0 {
				// When VCenters is 0, there should be no vcenter limit enforced
				// This is the default behavior - pools can be assigned from any vcenters
				if lease.Spec.VCenters != 0 {
					t.Error("VCenters should be 0 to indicate no limit")
				}
			}
		})
	}
}

// TestLeaseVCentersJSONSerialization tests JSON marshaling/unmarshaling
func TestLeaseVCentersJSONSerialization(t *testing.T) {
	original := &Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: LeaseSpec{
			VCpus:    16,
			Memory:   32,
			Pools:    3,
			VCenters: 2,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal lease: %v", err)
	}

	// Unmarshal back
	var unmarshaled Lease
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal lease: %v", err)
	}

	// Verify VCenters field is preserved
	if unmarshaled.Spec.VCenters != original.Spec.VCenters {
		t.Errorf("VCenters field not preserved after marshal/unmarshal: got %d, want %d",
			unmarshaled.Spec.VCenters, original.Spec.VCenters)
	}
}

// TestLeaseVCentersOmitEmpty tests that VCenters with value 0 is handled correctly
func TestLeaseVCentersOmitEmpty(t *testing.T) {
	lease := &Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: LeaseSpec{
			VCpus:    16,
			Memory:   32,
			VCenters: 0,
		},
	}

	data, err := json.Marshal(lease)
	if err != nil {
		t.Fatalf("Failed to marshal lease: %v", err)
	}

	// VCenters is marked as omitempty, so when it's 0 it may be omitted from JSON
	// But when unmarshaled, it should still be 0
	var unmarshaled Lease
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal lease: %v", err)
	}

	// Even if omitted in JSON, the zero value should be preserved
	if unmarshaled.Spec.VCenters != 0 {
		t.Errorf("Expected VCenters=0 after unmarshal, got %d", unmarshaled.Spec.VCenters)
	}
}

// TestLeaseMinVCentersValidation tests the validation marker for the MinVCenters field
// The actual validation happens in the CRD via kubebuilder:validation:Minimum=0
// These tests document expected behavior when creating/updating Lease objects
func TestLeaseMinVCentersValidation(t *testing.T) {
	tests := []struct {
		name        string
		minVcenters int
		expectValid bool
		description string
	}{
		{
			name:        "zero min-vcenters is valid (no minimum)",
			minVcenters: 0,
			expectValid: true,
			description: "When min-vcenters is 0 or unset, no vcenter minimum is applied",
		},
		{
			name:        "positive min-vcenters is valid",
			minVcenters: 1,
			expectValid: true,
			description: "Positive minimum values are valid",
		},
		{
			name:        "multiple min-vcenters is valid",
			minVcenters: 3,
			expectValid: true,
			description: "A minimum spanning several vcenters can be specified",
		},
		{
			name:        "large min-vcenters value is valid",
			minVcenters: 100,
			expectValid: true,
			description: "Large positive values are valid (satisfiability is evaluated by the controller)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lease := &Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lease",
					Namespace: "default",
				},
				Spec: LeaseSpec{
					VCpus:       16,
					Memory:      32,
					Pools:       3,
					MinVCenters: tt.minVcenters,
				},
			}

			// Marshal to JSON to ensure the field is serializable
			_, err := json.Marshal(lease)
			if err != nil && tt.expectValid {
				t.Errorf("Expected valid lease to marshal successfully, got error: %v", err)
			}

			// Verify the field value is set correctly
			if lease.Spec.MinVCenters != tt.minVcenters {
				t.Errorf("Expected MinVCenters=%d, got %d", tt.minVcenters, lease.Spec.MinVCenters)
			}
		})
	}
}

// TestLeaseMinVCentersDefaultValue tests that MinVCenters defaults to 0 when unset
func TestLeaseMinVCentersDefaultValue(t *testing.T) {
	lease := &Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: LeaseSpec{
			VCpus:  16,
			Memory: 32,
		},
	}

	// When MinVCenters is not explicitly set, it should default to 0
	if lease.Spec.MinVCenters != 0 {
		t.Errorf("Expected default MinVCenters=0, got %d", lease.Spec.MinVCenters)
	}
}

// TestLeaseMinVCentersWithPoolsField tests the interaction between Pools, VCenters,
// and MinVCenters
func TestLeaseMinVCentersWithPoolsField(t *testing.T) {
	tests := []struct {
		name        string
		pools       int
		vcenters    int
		minVcenters int
		description string
	}{
		{
			name:        "minimum equal to pools",
			pools:       3,
			vcenters:    0,
			minVcenters: 3,
			description: "Every pool must come from a different vcenter",
		},
		{
			name:        "minimum less than pools",
			pools:       4,
			vcenters:    0,
			minVcenters: 2,
			description: "4 pools must span at least 2 vcenters",
		},
		{
			name:        "minimum within cap",
			pools:       4,
			vcenters:    3,
			minVcenters: 2,
			description: "4 pools drawn from between 2 and 3 vcenters",
		},
		{
			name:        "minimum equal to cap means exact count",
			pools:       4,
			vcenters:    2,
			minVcenters: 2,
			description: "4 pools drawn from exactly 2 vcenters",
		},
		{
			name:        "zero minimum with multiple pools",
			pools:       5,
			vcenters:    0,
			minVcenters: 0,
			description: "No vcenter minimum - pools may all share one vcenter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lease := &Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lease",
					Namespace: "default",
				},
				Spec: LeaseSpec{
					VCpus:       16,
					Memory:      32,
					Pools:       tt.pools,
					VCenters:    tt.vcenters,
					MinVCenters: tt.minVcenters,
				},
			}

			// Verify all fields are set correctly
			if lease.Spec.Pools != tt.pools {
				t.Errorf("Expected Pools=%d, got %d", tt.pools, lease.Spec.Pools)
			}
			if lease.Spec.VCenters != tt.vcenters {
				t.Errorf("Expected VCenters=%d, got %d", tt.vcenters, lease.Spec.VCenters)
			}
			if lease.Spec.MinVCenters != tt.minVcenters {
				t.Errorf("Expected MinVCenters=%d, got %d", tt.minVcenters, lease.Spec.MinVCenters)
			}
		})
	}
}

// TestLeaseMinVCentersJSONSerialization tests JSON marshaling/unmarshaling
func TestLeaseMinVCentersJSONSerialization(t *testing.T) {
	original := &Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: LeaseSpec{
			VCpus:       16,
			Memory:      32,
			Pools:       3,
			MinVCenters: 2,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal lease: %v", err)
	}

	// The serialized form must use the min-vcenters JSON field name
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal lease into map: %v", err)
	}
	spec := raw["spec"].(map[string]interface{})
	if _, exists := spec["min-vcenters"]; !exists {
		t.Errorf("Expected 'min-vcenters' key in serialized spec, got: %v", spec)
	}

	// Unmarshal back
	var unmarshaled Lease
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal lease: %v", err)
	}

	// Verify MinVCenters field is preserved
	if unmarshaled.Spec.MinVCenters != original.Spec.MinVCenters {
		t.Errorf("MinVCenters field not preserved after marshal/unmarshal: got %d, want %d",
			unmarshaled.Spec.MinVCenters, original.Spec.MinVCenters)
	}
}

// TestLeaseMinVCentersOmitEmpty tests that MinVCenters with value 0 is handled correctly
func TestLeaseMinVCentersOmitEmpty(t *testing.T) {
	lease := &Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: LeaseSpec{
			VCpus:       16,
			Memory:      32,
			MinVCenters: 0,
		},
	}

	data, err := json.Marshal(lease)
	if err != nil {
		t.Fatalf("Failed to marshal lease: %v", err)
	}

	// MinVCenters is marked as omitempty, so when it's 0 it may be omitted from JSON
	// But when unmarshaled, it should still be 0
	var unmarshaled Lease
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal lease: %v", err)
	}

	// Even if omitted in JSON, the zero value should be preserved
	if unmarshaled.Spec.MinVCenters != 0 {
		t.Errorf("Expected MinVCenters=0 after unmarshal, got %d", unmarshaled.Spec.MinVCenters)
	}
}
