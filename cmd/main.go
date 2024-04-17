package main

import (
	"log"
	"net/http"

	_ "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/endpoints"
)

func main() {
	log.Fatal(http.ListenAndServe(":8080", nil))
}
