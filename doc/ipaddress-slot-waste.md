# IP Address Slot Waste in Network Resources

## Problem

Every Network resource's `ipAddresses` list reserves the first two indices for the network address and gateway:

| Index | Value | Purpose |
|-------|-------|---------|
| 0 | Network address (e.g., `.0` or `.128`) | Never assigned to a node |
| 1 | Gateway (e.g., `.1` or `.129`) | Never assigned to a node |
| 2 | First usable IP | LB / API VIP (UPI) or API VIP (IPI) |
| 3 | Second usable IP | Bootstrap (UPI) or Ingress VIP (IPI) |
| 4+ | Remaining IPs | Control plane nodes, then workers |

Both IPI and UPI scripts in `openshift/release` skip indices 0 and 1:

**IPI** (`ipi-conf-vsphere-vips-vcm-commands.sh`):
```bash
jq -r --argjson N 2 '.spec.ipAddresses[$N]'   # API VIP
jq -r --argjson N 3 '.spec.ipAddresses[$N]'   # Ingress VIP
```

**UPI** (`upi-conf-vsphere-vcm-commands.sh`):
```bash
lb_ip_address=$(jq -r '.spec.ipAddresses[2]' "${NETWORK_CONFIG}")
bootstrap_ip_address=$(jq -r '.spec.ipAddresses[3]' "${NETWORK_CONFIG}")
# control plane starts at index 4, workers after that
```

The network address and gateway are redundant here — they're already available via the `gateway` and `machineNetworkCidr` fields on the Network spec.

## Impact

For single-tenant networks with 20 IPs, losing 2 slots is a ~10% overhead — not significant.

For multi-tenant networks with sliding windows of 4-5 IPs, losing 2 slots to sentinels means only 2-3 usable addresses — **40-50% waste**.

## Options

### Option 1: Remove sentinel entries from `ipAddresses`

Stop including the network address and gateway in `ipAddresses`. Start the list at what is currently index 2 (the first assignable IP). Update the IPI and UPI scripts to consume starting at index 0 instead of index 2.

- Requires a coordinated change across `vsphere-capacity-manager` (network definitions) and `openshift/release` (step registry scripts).
- Reclaims 2 IPs per network.
- Cleaner long-term: `ipAddresses` contains only addresses that are actually assignable.

### Option 2: Accept the overhead

Keep the current convention. Size networks with 2 extra IPs to compensate.

- Zero code changes.
- Every new network continues to burn 2 IPs.
- Multi-tenant sliding windows remain constrained.
