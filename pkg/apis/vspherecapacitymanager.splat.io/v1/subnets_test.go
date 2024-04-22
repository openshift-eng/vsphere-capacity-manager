package v1

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSubnetCalculation(t *testing.T) {
	content, err := os.ReadFile("/home/rvanderp/code/vsphere-capacity-manager/subnets.json")
	if err != nil {
		t.Fatalf("error reading subnets.json: %s", err)
	}

	var subnets Subnets
	err = json.Unmarshal(content, &subnets)
	if err != nil {
		t.Fatalf("error unmarshalling subnets.json: %s", err)
	}
}
