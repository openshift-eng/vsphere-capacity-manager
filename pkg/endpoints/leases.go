package endpoints

import (
	"encoding/json"
	"log"
	"net/http"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/resources"
)

func init() {
	log.Printf("initializing endpoints")
	http.HandleFunc("/acquire", acquireLeaseHandler)
	http.HandleFunc("/show-pools", showPoolStatusHandler)
	http.HandleFunc("/release", releaseLeaseHandler)
}

func releaseLeaseHandler(w http.ResponseWriter, r *http.Request) {
	var res v1.ResourceRequest
	err := json.NewDecoder(r.Body).Decode(&res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func acquireLeaseHandler(w http.ResponseWriter, r *http.Request) {
	var res v1.ResourceRequest
	err := json.NewDecoder(r.Body).Decode(&res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	leases, err := resources.AcquireLease(&res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	marshalledLeases, err := json.Marshal(leases)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(marshalledLeases)
}

func showPoolStatusHandler(w http.ResponseWriter, r *http.Request) {
	pools := resources.GetPools()
	marshalledPools, err := json.Marshal(pools)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(marshalledPools)
}
