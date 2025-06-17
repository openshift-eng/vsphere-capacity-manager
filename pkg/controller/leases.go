package controller

import (
	"context"
	"fmt"
	"log"
	"path"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/utils"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/utils/conditions"
)

const (
	BoskosIdLabel             = "boskos-lease-id"
	JobNameLabel              = "job-name"
	ALLOW_MULTI_TO_USE_SINGLE = false

	// PROW_JOB_PERIODICAL_URL is used to generate URL for periodical jobs.  Need to supply PROW_JOB_URL_PREFIX_KEY, PROW_GS_BUCKET_KEY, PROW_JOB and PROW_BUILD_ID.
	PROW_JOB_PERIODICAL_URL = "%vgs/%v/logs/%v/%v"

	// PROW_JOB_PRESUBMIT_URL is used to generate URL for presubmit jobs.  Need to supply PROW_JOB_URL_PREFIX_KEY, PROW_GS_BUCKET_KEY,GIT_ORG, GIT_REPO, GIT_PR, PROW_JOB, and PROW_BUILD_ID.
	PROW_JOB_PRESUBMIT_URL = "%vgs/%v/pr-logs/pull/%v_%v/%v/%v/%v"

	PROW_JOB_TYPE_KEY       = "prow-job-type"
	PROW_JOB_KEY            = "prow-job-name"
	PROW_JOB_URL_PREFIX_KEY = "prow-url-prefix"
	PROW_GS_BUCKET_KEY      = "prow-gs-bucket"
	PROW_BUILD_ID_KEY       = "prow-build-id"
	GIT_ORG_KEY             = "git-org"
	GIT_REPO_KEY            = "git-repo"
	GIT_PR_KEY              = "git-pr"

	DEFAULT_PROW_JOB_URL_PREFIX = "https://prow.ci.openshift.org/view/"
	DEFAULT_PROW_GS_BUCKET      = "test-platform-results"
	PERIODICAL_JOB_TYPE         = "periodic"
	PRESUBMIT_JOB_TYPE          = "presubmit"
)

type LeaseReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
	RESTMapper     meta.RESTMapper
	UncachedClient client.Client

	// Namespace is the namespace in which the ControlPlaneMachineSet controller should operate.
	// Any ControlPlaneMachineSet not in this namespace should be ignored.
	Namespace string

	// OperatorName is the name of the ClusterOperator with which the controller should report
	// its status.
	OperatorName string

	// ReleaseVersion is the version of current cluster operator release.
	ReleaseVersion string

	// Option to allow multi-tenant lease to use single-tenant networks
	AllowMultiToUseSingle bool
}

func (l *LeaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.Lease{}).
		Complete(l); err != nil {
		return fmt.Errorf("error setting up controller: %w", err)
	}

	// Set up API helpers from the manager.
	l.Client = mgr.GetClient()
	l.Scheme = mgr.GetScheme()
	l.Recorder = mgr.GetEventRecorderFor("leases-controller")
	l.RESTMapper = mgr.GetRESTMapper()

	leases = make(map[string]*v1.Lease)
	pools = make(map[string]*v1.Pool)
	networks = make(map[string]*v1.Network)
	return nil
}

// getNetworksForPool get all networks for the provided pool.
func getNetworksForPool(pool *v1.Pool) map[string]*v1.Network {
	networksInPool := make(map[string]*v1.Network)
	for _, portGroupPath := range pool.Spec.Topology.Networks {
		_, networkName := path.Split(portGroupPath)

		for _, network := range networks {
			if (*network.Spec.PodName == pool.Spec.IBMPoolSpec.Pod) &&
				(network.Spec.PortGroupName == networkName) {
				networksInPool[network.Name] = network
				break
			}
		}
	}
	return networksInPool
}

// getAvailableNetworks retrieves networks which are not owned by a lease
func (l *LeaseReconciler) getAvailableNetworks(pool *v1.Pool, networkType v1.NetworkType) []*v1.Network {
	networksInPool := getNetworksForPool(pool)
	availableNetworks := make([]*v1.Network, 0)

	for _, network := range networksInPool {
		hasOwner := false
		for _, lease := range leases {
			for _, ownerRef := range lease.OwnerReferences {
				if ownerRef.Name == network.Name &&
					ownerRef.Kind == network.Kind {
					hasOwner = true
					break
				}
			}
			if hasOwner {
				break
			}
		}

		thisNetworkType := string(v1.NetworkTypeSingleTenant)
		if network.ObjectMeta.Labels != nil {
			if val, exists := network.ObjectMeta.Labels[v1.NetworkTypeLabel]; exists {
				log.Printf("network found with NeworkTypeLabel: %s", val)
				thisNetworkType = val
			}
		}
		if thisNetworkType != string(networkType) {
			continue
		}
		if !hasOwner {
			availableNetworks = append(availableNetworks, network)
		}
	}
	return availableNetworks
}

func getIBMDatacenterAndPod(server string) (string, string) {
	for _, pool := range pools {
		if pool.Spec.Server == server {
			return pool.Spec.IBMPoolSpec.Datacenter, pool.Spec.IBMPoolSpec.Pod
		}
	}
	return "", ""
}

// reconcilePoolStates updates the states of all pools. this ensures we have the most up-to-date state of the pools
// before we attempt to reconcile any leases. the pool resource statuses are not updated.
func reconcilePoolStates() []*v1.Pool {
	var outList []*v1.Pool

	networksInUse := make(map[string]map[string]string)

	for poolName, pool := range pools {
		vcpus := 0
		memory := 0
		leaseCount := 0

		for _, lease := range leases {
			for _, ownerRef := range lease.OwnerReferences {
				if ownerRef.Kind == pool.Kind && ownerRef.Name == pool.Name {
					vcpus += lease.Spec.VCpus
					memory += lease.Spec.Memory
					leaseCount++

					var serverNetworks map[string]string
					var exists bool

					dc, pod := getIBMDatacenterAndPod(lease.Status.Server)
					dcId := fmt.Sprintf("dcid-%s-%s", dc, pod)
					if serverNetworks, exists = networksInUse[dcId]; !exists {
						serverNetworks = make(map[string]string)
						networksInUse[dcId] = serverNetworks
					}

					for _, networkPath := range lease.Status.Topology.Networks {
						_, networkName := path.Split(networkPath)
						serverNetworks[networkName] = networkName
					}
					break
				}
			}
		}

		overCommitRatio, err := strconv.ParseFloat(pool.Spec.OverCommitRatio, 32)
		if err != nil {
			log.Printf("error converting overCommitRatio to float %v setting to 1.0", err)
			overCommitRatio = 1.0
		}

		pool.Status.VCpusAvailable = int(float64(pool.Spec.VCpus)*overCommitRatio) - vcpus
		pool.Status.MemoryAvailable = pool.Spec.Memory - memory
		pool.Status.LeaseCount = leaseCount

		pools[poolName] = pool
		outList = append(outList, pool)
	}

	for _, pool := range outList {
		availableNetworks := 0
		for _, network := range pool.Spec.Topology.Networks {
			_, networkName := path.Split(network)
			dcId := fmt.Sprintf("dcid-%s-%s", pool.Spec.IBMPoolSpec.Datacenter, pool.Spec.IBMPoolSpec.Pod)
			serverNetworks := networksInUse[dcId]
			if _, ok := serverNetworks[networkName]; !ok {
				availableNetworks++
			}
		}
		pool.Status.NetworkAvailable = availableNetworks
	}

	return outList
}

func (l *LeaseReconciler) triggerPoolUpdates(ctx context.Context) {
	for _, pool := range pools {

		err := l.Client.Get(ctx, types.NamespacedName{Name: pool.Name, Namespace: pool.Namespace}, pool)
		if err != nil {
			log.Printf("error getting pool %s: %v", pool.Name, err)
			continue
		}

		if pool.Annotations == nil {
			pool.Annotations = make(map[string]string)
		}

		pool.Annotations["last-updated"] = time.Now().Format(time.RFC3339)
		err = l.Client.Update(ctx, pool)
		if err != nil {
			log.Printf("error updating pool %s annotations: %v", pool.Name, err)
		}
	}
}

func (l *LeaseReconciler) triggerLeaseUpdates(ctx context.Context, networkType v1.NetworkType) {
	var oldestLease *v1.Lease
	for _, lease := range leases {
		// If networkType doesn't match desired, then skip
		if lease.Spec.NetworkType != networkType {
			continue
		}

		// We only want to force an update for leases that are Pending or Partial
		if lease.Status.Phase == v1.PHASE_FULFILLED {
			continue
		}

		err := l.Client.Get(ctx, types.NamespacedName{Name: lease.Name, Namespace: lease.Namespace}, lease)
		if err != nil {
			log.Printf("error getting lease %s: %v", lease.Name, err)
			continue
		}

		// If lease creation time is older than oldestLease, make current lease the oldestLease
		if oldestLease == nil || lease.CreationTimestamp.Before(&oldestLease.CreationTimestamp) {
			oldestLease = lease
		}

	}

	if oldestLease != nil {
		if oldestLease.Annotations == nil {
			oldestLease.Annotations = make(map[string]string)
		}

		log.Printf("triggering lease update %v", oldestLease.Name)
		oldestLease.Annotations["last-updated"] = time.Now().Format(time.RFC3339)
		err := l.Client.Update(ctx, oldestLease)
		if err != nil {
			log.Printf("error updating lease %s annotations: %v", oldestLease.Name, err)
		}
	}
}

func updateLeaseMetrics() {
	LeaseCounts.Reset()
	for _, lease := range leases {
		promLabels := make(prometheus.Labels)
		promLabels["phase"] = string(lease.Status.Phase)
		promLabels["networkType"] = string(lease.Spec.NetworkType)
		promLabels["namespace"] = lease.Namespace

		LeaseCounts.With(promLabels).Inc()
	}
}

// returns common portgroups that satisfies all known leases for this job. common port groups are scoped
// to a single vCenter. for multiple vCenters, a network lease for each vCenter will be claimed.
func (l *LeaseReconciler) getCommonNetworksForLease(lease *v1.Lease) ([]*v1.Network, error) {
	var exists bool
	var leaseID string

	if lease.Spec.VCpus == 0 && lease.Spec.Memory == 0 {
		return nil, fmt.Errorf("network-only lease %s", lease.Name)
	}
	if leaseID, exists = lease.Labels[BoskosIdLabel]; !exists {
		return nil, fmt.Errorf("no lease label found for %s", lease.Name)
	}

	for _, _lease := range leases {
		if _lease.Spec.VCpus == 0 && _lease.Spec.Memory == 0 {
			// this is a network-only lease. do not consider it.
			continue
		}

		if thisLeaseID, exists := _lease.Labels[BoskosIdLabel]; !exists {
			continue
		} else if thisLeaseID != leaseID {
			continue
		} else if lease.Status.Phase != v1.PHASE_PENDING {
			continue
		}

		var foundNetworks []*v1.Network
		for _, ownerRef := range _lease.OwnerReferences {
			if ownerRef.Kind != "Network" {
				continue
			}

			// If the lease is requiring more than one, we need to return all that fulfill the request.  Multi nic
			// fails here if the network count is 2 and we return 1.
			for _, network := range networks {
				if network.Name == ownerRef.Name && network.UID == ownerRef.UID {
					foundNetworks = append(foundNetworks, network)
				}
			}
		}
		if len(foundNetworks) > 0 {
			return foundNetworks, nil
		}
	}
	return nil, fmt.Errorf("no common network found for %s", lease.Name)
}

// shouldLeaseBeDelayed is used to determine if current lease should be delayed.
func shouldLeaseBeDelayed(lease *v1.Lease) bool {
	// Iterate through all leases.  Ignore fulfilled.  If we see Partial, block if needing same pool.  If Pending, we
	// can only run if there are no other partials that are interested in the same pools as current lease.  If there are
	// no partials, then we need to make sure we have no other leases that are older.  Oldest should go first.
	if lease.Status.Phase == v1.PHASE_PENDING {
		for _, curLease := range leases {

			// skip if lease is the target lease
			if curLease.Name == lease.Name {
				continue
			}

			// If the lease type does not match, continue
			if curLease.Spec.NetworkType != lease.Spec.NetworkType {
				continue
			}

			// If lease is multi network and required pool is blank, then we want to make sure the current assigned pool
			// is checked instead of desired pool.
			requiredPool := curLease.Spec.RequiredPool
			if curLease.Spec.Networks > 1 && requiredPool == "" {
				requiredPool = curLease.Status.Name
			}

			log.Printf("lease %v pool '%v', current lease %v (%v) pool '%v'", lease.Name, lease.Spec.RequiredPool, curLease.Name, curLease.Status.Phase, requiredPool)

			switch curLease.Status.Phase {
			case v1.PHASE_FULFILLED:
				continue
			case v1.PHASE_PARTIAL:
				// We want partial to prevent others wanting same pool.
				if requiredPool == lease.Spec.RequiredPool || lease.Spec.RequiredPool == "" {
					return true
				}
			case v1.PHASE_PENDING:
				// If leases are both from the same pool, give priority to oldest.  If either of them are blank for the
				// desired pool, they could be assigned to the same pool depending on availability.  So in this case,
				// compare them as well.
				if requiredPool == lease.Spec.RequiredPool || requiredPool == "" || lease.Spec.RequiredPool == "" {
					leaseTime := curLease.CreationTimestamp
					if leaseTime.Time.Before(lease.CreationTimestamp.Time) {
						return true
					}
				}
			default:
				log.Printf("unknown lease phase %s", curLease.Status.Phase)
			}
		}
	}
	return false
}

// doesLeaseContainPortGroup checks to see if the supplied network is part of a portgroup that is already assigned to the lease.
func doesLeaseContainPortGroup(lease *v1.Lease, pool *v1.Pool, network *v1.Network) bool {
	poolNetworks := getNetworksForPool(pool)

	for _, owner := range lease.OwnerReferences {
		if owner.Kind == "Network" {
			if poolNetworks[owner.Name].Spec.VlanId == network.Spec.VlanId &&
				*poolNetworks[owner.Name].Spec.DatacenterName == *network.Spec.DatacenterName {
				return true
			}
		}
	}

	return false
}

func generateJobLink(lease *v1.Lease) string {
	jobURL := ""
	if lease.Annotations != nil {
		jobURLPrefix := lease.Annotations[PROW_JOB_URL_PREFIX_KEY]
		if jobURLPrefix == "" {
			jobURLPrefix = DEFAULT_PROW_JOB_URL_PREFIX
		}

		prowGSBucket := lease.Annotations[PROW_GS_BUCKET_KEY]
		if prowGSBucket == "" {
			prowGSBucket = DEFAULT_PROW_GS_BUCKET
		}

		switch lease.Annotations[PROW_JOB_TYPE_KEY] {
		case PERIODICAL_JOB_TYPE:
			jobURL = fmt.Sprintf(PROW_JOB_PERIODICAL_URL, jobURLPrefix, prowGSBucket, lease.Annotations[PROW_JOB_KEY], lease.Annotations[PROW_BUILD_ID_KEY])
		case PRESUBMIT_JOB_TYPE:
			jobURL = fmt.Sprintf(PROW_JOB_PRESUBMIT_URL, jobURLPrefix, prowGSBucket, lease.Annotations[GIT_ORG_KEY], lease.Annotations[GIT_REPO_KEY], lease.Annotations[GIT_PR_KEY], lease.Annotations[PROW_JOB_KEY], lease.Annotations[PROW_BUILD_ID_KEY])
		default:
			log.Printf("unknown job type: %v", lease.Annotations[PROW_JOB_TYPE_KEY])
		}
	} else {
		log.Printf("Unable to generate job url for lease %v due to missing annotations", lease.Name)
	}
	return jobURL
}

func (l *LeaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	reconcileLock.Lock()
	defer reconcileLock.Unlock()

	log.Print("Reconciling lease")
	defer log.Print("Finished reconciling lease")

	leaseKey := fmt.Sprintf("%s/%s", req.Namespace, req.Name)
	// Fetch the Lease instance.
	lease := &v1.Lease{}
	if err := l.Get(ctx, req.NamespacedName, lease); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if len(lease.Status.Phase) == 0 {
		lease.Status.Phase = v1.PHASE_PENDING
		lease.Status.Topology.Datacenter = "pending"
		lease.Status.Topology.Datastore = "/pending/datastore/pending"
		lease.Status.Topology.ComputeCluster = "/pending/host/pending"
		lease.Status.Server = "pending"
		lease.Status.Zone = "pending"
		lease.Status.Region = "pending"
		lease.Status.Name = "pending"
		lease.Status.ShortName = "pending"
		lease.Status.Topology.Networks = append(lease.Status.Topology.Networks, "/pending/network/pending")

		// Add the job link / info to status field.
		lease.Status.JobLink = generateJobLink(lease)
		log.Printf("generated job url '%v' for lease '%v'", lease.Status.JobLink, lease.Name)

		conditions.Set(lease, conditions.FalseCondition(
			v1.LeaseConditionTypeFulfilled,
		))
		conditions.Set(lease, conditions.FalseCondition(
			v1.LeaseConditionTypeDelayed,
		))
		conditions.Set(lease, conditions.TrueCondition(
			v1.LeaseConditionTypePending,
		))
		conditions.Set(lease, conditions.FalseCondition(
			v1.LeaseConditionTypePartial,
		))

		if err := l.Status().Update(ctx, lease); err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to set the initial status on the lease %s: %w", lease.Name, err)
		}
	}

	if lease.Finalizers == nil {
		log.Print("setting finalizer on lease")
		lease.Finalizers = []string{v1.LeaseFinalizer}
		err := l.Client.Update(ctx, lease)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error setting lease finalizer: %w", err)
		}
	}

	promLabels := make(prometheus.Labels)
	promLabels["namespace"] = req.Namespace

	if lease.DeletionTimestamp != nil {
		log.Printf("lease %s is being deleted at %s", lease.Name, lease.DeletionTimestamp.String())

		// preserve finalizers not associated with VCM
		if lease.Finalizers != nil {
			var preservedFinalizers []string

			for _, finalizer := range lease.Finalizers {
				if finalizer == v1.LeaseFinalizer {
					continue
				}

				preservedFinalizers = append(preservedFinalizers, finalizer)
			}
			lease.Finalizers = preservedFinalizers
		}

		err := l.Update(ctx, lease)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error dropping finalizers from lease: %w", err)
		}

		if ownRef := utils.DoesLeaseHavePool(lease); ownRef != nil {
			promLabels["pool"] = ownRef.Name
		}

		delete(leases, leaseKey)
		if len(promLabels) >= 2 {
			LeasesInUse.With(promLabels).Dec()
		}
		reconcilePoolStates()
		l.triggerPoolUpdates(ctx)
		l.triggerLeaseUpdates(ctx, lease.Spec.NetworkType)
		updateLeaseMetrics()
		return ctrl.Result{}, nil
	}

	leases[leaseKey] = lease

	if lease.Status.Phase == v1.PHASE_FULFILLED {
		log.Print("lease is already fulfilled")
		return ctrl.Result{}, nil
	}

	updatedPools := reconcilePoolStates()

	// TODO: How often are we hitting this and can we remove this and just use the one above?
	if len(lease.Status.Phase) == 0 {
		log.Printf("setting lease %s status to %s", lease.Name, v1.PHASE_PENDING)
		lease.Status.Phase = v1.PHASE_PENDING

		conditions.Set(lease, conditions.FalseCondition(
			v1.LeaseConditionTypeFulfilled,
		))
		conditions.Set(lease, conditions.FalseCondition(
			v1.LeaseConditionTypeDelayed,
		))
		conditions.Set(lease, conditions.TrueCondition(
			v1.LeaseConditionTypePending,
		))
		conditions.Set(lease, conditions.FalseCondition(
			v1.LeaseConditionTypePartial,
		))
	}

	jobName, exists := lease.Labels[JobNameLabel]
	if !exists {
		jobName = "Unknown Job"
	}
	log.Printf("processing lease %v [%v] with Phase %v", lease.Name, jobName, lease.Status.Phase)

	// Set default network type
	if len(lease.Spec.NetworkType) == 0 {
		lease.Spec.NetworkType = v1.NetworkTypeSingleTenant
	}

	// We need to check to see if any other leases are waiting for resources that this lease may want.  We need to
	// ensure that older leases get to finish getting their requests fulfilled before their Ci jobs timeout.
	if shouldLeaseBeDelayed(lease) {
		log.Printf("=========== lease %v is being delayed due to presence of higher priority leases ===========", lease.Name)

		conditions.Set(lease, conditions.TrueCondition(
			v1.LeaseConditionTypeDelayed,
		))
		conditions.Set(lease, conditions.FalseConditionWithReason(
			v1.LeaseConditionTypeFulfilled,
			v1.ReasonLeaseDelayed,
			v1.ConditionSeverityInfo,
			"lease is being delayed due to presence of higher priority leases",
		))

		if err := l.Client.Status().Update(ctx, lease); err != nil {
			return reconcile.Result{}, err
		}

		// Since we are delaying this lease, let's force the oldest lease to be updated to see if it can now be fulfilled.
		l.triggerLeaseUpdates(ctx, lease.Spec.NetworkType)
		updateLeaseMetrics()

		return ctrl.Result{}, fmt.Errorf("lease %v is being delayed", lease.Name)
	}

	// Since lease is not delayed, clear the condition
	conditions.Set(lease, conditions.FalseCondition(
		v1.LeaseConditionTypeDelayed,
	))

	pool := &v1.Pool{}
	if ref := utils.DoesLeaseHavePool(lease); ref == nil {
		pool, err = utils.GetPoolWithStrategy(lease, updatedPools, v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
		if err != nil {
			conditions.Set(lease, conditions.FalseConditionWithReason(
				v1.LeaseConditionTypeFulfilled,
				v1.ReasonLeaseNoPool,
				v1.ConditionSeverityWarning,
				err.Error(),
			))

			if uErr := l.Client.Status().Update(ctx, lease); uErr != nil {
				log.Printf("unable to update lease: %v", uErr)
			}

			// since we do not trigger lease update, we still need to update metrics in case first status update.
			updateLeaseMetrics()
			return ctrl.Result{}, fmt.Errorf("unable to get matching pool: %v", err)
		}
	} else {
		err = l.Get(ctx, types.NamespacedName{
			Namespace: req.Namespace,
			Name:      ref.Name,
		}, pool)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error getting pool: %v", err)
		}
	}

	var network *v1.Network

	if !utils.DoesLeaseHaveNetworks(lease) {
		log.Printf("Searching for networks to assign to lease %v", lease.Name)
		var availableNetworks []*v1.Network
		availableNetworks, err = l.getCommonNetworksForLease(lease)
		if err != nil {
			log.Printf("error getting common network for lease, will attempt to allocate a new one: %v", err)

			availableNetworks = l.getAvailableNetworks(pool, lease.Spec.NetworkType)

			// We can allow multi-tenant leases to use single-tenant networks if there are not enough multi-tenant leases.
			if l.AllowMultiToUseSingle && lease.Spec.NetworkType == v1.NetworkTypeMultiTenant {
				// for mutli-tenant leases, there is no reason they can't fall back to single-tenant if there aren't
				// any multi-tenant leases available.
				log.Println("Adding single tenant to multi tenant collection...")
				availableNetworks = append(availableNetworks, l.getAvailableNetworks(pool, v1.NetworkTypeSingleTenant)...)
			}
		} else {
			log.Printf("getCommonNetworkForLease for lease %v returned %d leases", lease.Name, len(availableNetworks))
		}

		log.Printf("available networks: %d - lease %s requested %d networks and current has %d assigned", len(availableNetworks), lease.Name, lease.Spec.Networks, len(lease.Status.Topology.Networks))

		// If we do not have enough networks, lets assign the ones we do have and assign additional when they become available to prevent starvation.
		if len(availableNetworks) == 0 {
			return ctrl.Result{}, fmt.Errorf("lease requires %d networks, %d networks available", lease.Spec.Networks, len(availableNetworks))
		}

		// Set networks to equal current ones in status
		var networks []string
		if len(lease.Status.Topology.Networks) != 0 && lease.Status.Topology.Networks[0] != "/pending/network/pending" {
			networks = lease.Status.Topology.Networks
		}

		for idx := 0; idx+len(lease.Status.Topology.Networks) < lease.Spec.Networks && idx < len(availableNetworks); idx++ {
			if !doesLeaseContainPortGroup(lease, pool, availableNetworks[idx]) {
				network = availableNetworks[idx]
				lease.OwnerReferences = append(lease.OwnerReferences, metav1.OwnerReference{
					APIVersion: network.APIVersion,
					Kind:       network.Kind,
					Name:       network.Name,
					UID:        network.UID,
				})
				networks = append(networks, fmt.Sprintf("/%s/network/%s", lease.Status.Topology.Datacenter, network.Spec.PortGroupName))
			}
		}
		if len(networks) != lease.Spec.Networks {
			log.Printf("%s requested more than one network, but only %d have been assigned", lease.Name, len(networks))
		}
		lease.Status.Topology.Networks = networks
	}

	// This is currently setting last network as env.  I believe it shouldn't matter, but may want to use the first one.
	if network != nil {
		log.Printf("Generating env vars for lease %v with pool %v and network %v", lease.Name, pool.Name, network.Name)
		err = utils.GenerateEnvVars(lease, pool, network)
		if err != nil {
			log.Printf("error generating env vars: %v", err)
		}
	}

	// If all networks have been assigned, lets mark lease as Fulfilled, else the phase will be partial
	log.Printf("Lease %v has %d networks assigned: %v", lease.Name, len(lease.Status.Topology.Networks), lease.Status.Topology.Networks)
	if len(lease.Status.Topology.Networks) == lease.Spec.Networks {
		lease.Status.Phase = v1.PHASE_FULFILLED

		conditions.Set(lease, conditions.TrueCondition(
			v1.LeaseConditionTypeFulfilled,
		))
		conditions.Set(lease, conditions.FalseCondition(
			v1.LeaseConditionTypePending,
		))
		conditions.Set(lease, conditions.FalseCondition(
			v1.LeaseConditionTypePartial,
		))
	} else {
		lease.Status.Phase = v1.PHASE_PARTIAL
		conditions.Set(lease, conditions.FalseCondition(
			v1.LeaseConditionTypePending,
		))
		conditions.Set(lease, conditions.FalseConditionWithReason(
			v1.LeaseConditionTypeFulfilled,
			v1.ReasonLeasePartial,
			v1.ConditionSeverityInfo,
			fmt.Sprintf("lease is currently assigned %v of %v networks", len(lease.Status.Topology.Networks), lease.Spec.Networks),
		))
		conditions.Set(lease, conditions.TrueCondition(
			v1.LeaseConditionTypePartial,
		))
	}

	leaseStatus := lease.Status.DeepCopy()
	err = l.Client.Update(ctx, lease)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating lease, requeuing: %v", err)
	}

	leaseStatus.DeepCopyInto(&lease.Status)

	err = l.Client.Status().Update(ctx, lease)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating lease, requeuing: %v", err)
	}

	if lease.Status.Phase == v1.PHASE_FULFILLED {
		promLabels["pool"] = pool.Name
		LeasesInUse.With(promLabels).Add(1)

		if pool.Annotations == nil {
			pool.Annotations = make(map[string]string)
		}

		l.triggerPoolUpdates(ctx)
		l.triggerLeaseUpdates(ctx, lease.Spec.NetworkType)
	}
	updateLeaseMetrics()

	return ctrl.Result{}, nil
}
