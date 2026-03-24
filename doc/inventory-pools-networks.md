# Pools and networks inventory

This file is a **point-in-time snapshot** of custom resources in the management cluster. It is not regenerated automatically when the cluster changes.

| Field | Value |
|--------|--------|
| Captured (UTC) | 2026-03-24 12:50 |
| `kubectl` / `oc` context | `default/api-build02-vmc-ci-openshift-org:6443/jcallen` |
| Namespace | `vsphere-infra-helpers` |

## Pools

| Name | vCenter (`spec.server`) | Port groups in spec | Excluded | NoSchedule | Taints | vCPUs avail | Memory (GB) avail | Networks avail | Leases |
|------|---------------------------|---------------------|----------|------------|--------|-------------|------------------|----------------|--------|
| `vcenter-1.ci.ibmc.devcluster.openshift.com-cidatacenter-2-cicluster-3` | vcenter-1.ci.ibmc.devcluster.openshift.com | 69 | false | false | 0 | 416 | 3200 | 34 | 10 |
| `vcenter-1.devqe.ibmc.devcluster.openshift.com-devqedatacenter-1-devqecluster-1` | vcenter-1.devqe.ibmc.devcluster.openshift.com | 18 | true | false | 0 | 256 | 2047 | 18 | 0 |
| `vcenter-110.ci.ibmc.devcluster.openshift.com-vcenter-110-dc01-vcenter-110-cl01` | vcenter-110.ci.ibmc.devcluster.openshift.com | 73 | false | false | 0 | 240 | 3520 | 38 | 6 |
| `vcenter-120.ci.ibmc.devcluster.openshift.com-wldn-120-dc-wldn-120-cl01` | vcenter-120.ci.ibmc.devcluster.openshift.com | 61 | false | false | 0 | 235 | 3808 | 29 | 3 |
| `vcenter-130.ci.ibmc.devcluster.openshift.com-wldn-130-dc-wldn-130-cl01` | vcenter-130.ci.ibmc.devcluster.openshift.com | 61 | true | false | 0 | 422 | 4096 | 29 | 0 |
| `vcenter-7-nested-dal10.pod03` | 10.93.60.135 | 4 | true | false | 0 | 96 | 384 | 3 | 0 |
| `vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-1-cicluster-1` | vcenter.ci.ibmc.devcluster.openshift.com | 69 | false | false | 0 | 180 | 719 | 34 | 5 |
| `vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-1-cicluster-2` | vcenter.ci.ibmc.devcluster.openshift.com | 70 | false | false | 0 | 176 | 703 | 35 | 4 |
| `vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-cicluster` | vcenter.ci.ibmc.devcluster.openshift.com | 70 | false | false | 0 | 372 | 3024 | 35 | 12 |
| `vcenter.devqe.ibmc.devcluster.openshift.com-devqedatacenter-devqecluster` | vcenter.devqe.ibmc.devcluster.openshift.com | 18 | true | true | 0 | 192 | 3070 | 18 | 0 |

## Networks

| Name | Port group | VLAN | Pod | Datacenter | Machine CIDR | `network-type` label |
|------|------------|------|-----|--------------|--------------|----------------------|
| `ci-vlan-1108-1-dal10-dal10.pod03` | ci-vlan-1108 | 1108 | dal10.pod03 | dal10 | 10.221.127.0/25 | nested-multi-tenant |
| `ci-vlan-1108-1-dal10-dal10.pod03-1` | ci-vlan-1108-1 | 1108 | dal10.pod03 | dal10 | 10.221.127.0/25 | multi-tenant |
| `ci-vlan-1108-1-dal10-dal10.pod03-2` | ci-vlan-1108-2 | 1108 | dal10.pod03 | dal10 | 10.221.127.0/25 | multi-tenant |
| `ci-vlan-1108-1-dal10-dal10.pod03-3` | ci-vlan-1108-3 | 1108 | dal10.pod03 | dal10 | 10.221.127.0/25 | multi-tenant |
| `ci-vlan-1108-1-dal10-dal10.pod03-4` | ci-vlan-1108-4 | 1108 | dal10.pod03 | dal10 | 10.221.127.0/25 | multi-tenant |
| `ci-vlan-1140-dal12-dal12.pod01` | ci-vlan-1140 | 1140 | dal12.pod01 | dal12 | 10.241.55.128/25 | single-tenant |
| `ci-vlan-1146-dal12-dal12.pod01` | ci-vlan-1146 | 1146 | dal12.pod01 | dal12 | 10.241.82.0/25 | single-tenant |
| `ci-vlan-1146-disconneted-dal12-dal12.pod01` | ci-vlan-1146-disconneted | 1146 | dal12.pod01 | dal12 | 10.241.82.0/25 | single-tenant |
| `ci-vlan-1148-dal10-dal10.pod03` | ci-vlan-1148 | 1148 | dal10.pod03 | dal10 | 10.93.43.128/25 | nested-multi-tenant |
| `ci-vlan-1148-dal10-dal10.pod03-2` | ci-vlan-1148-1 | 1148 | dal10.pod03 | dal10 | 10.93.43.128/25 | multi-tenant |
| `ci-vlan-1148-dal10-dal10.pod03-3` | ci-vlan-1148-2 | 1148 | dal10.pod03 | dal10 | 10.93.43.128/25 | multi-tenant |
| `ci-vlan-1148-dal10-dal10.pod03-4` | ci-vlan-1148-3 | 1148 | dal10.pod03 | dal10 | 10.93.43.128/25 | multi-tenant |
| `ci-vlan-1148-dal10-dal10.pod03-5` | ci-vlan-1148-4 | 1148 | dal10.pod03 | dal10 | 10.93.43.128/25 | multi-tenant |
| `ci-vlan-1148-dal12-dal12.pod01` | ci-vlan-1148 | 1148 | dal12.pod01 | dal12 | 10.241.103.0/25 | single-tenant |
| `ci-vlan-1154-dal12-dal12.pod01` | ci-vlan-1154 | 1154 | dal12.pod01 | dal12 | 10.241.96.128/25 | single-tenant |
| `ci-vlan-1158-dal12-dal12.pod01` | ci-vlan-1158 | 1158 | dal12.pod01 | dal12 | 10.184.20.128/25 | single-tenant |
| `ci-vlan-1161-dal10-dal10.pod03` | ci-vlan-1161 | 1161 | dal10.pod03 | dal10 | 10.93.63.128/25 | single-tenant |
| `ci-vlan-1164-dal12-dal12.pod01` | ci-vlan-1164 | 1164 | dal12.pod01 | dal12 | 10.241.111.128/25 | single-tenant |
| `ci-vlan-1164-disconneted-dal12-dal12.pod01` | ci-vlan-1164-disconneted | 1164 | dal12.pod01 | dal12 | 10.241.111.128/25 | single-tenant |
| `ci-vlan-1165-dal12-dal12.pod01` | ci-vlan-1165 | 1165 | dal12.pod01 | dal12 | 10.241.95.128/25 | single-tenant |
| `ci-vlan-1166-dal12-dal12.pod01` | ci-vlan-1166 | 1166 | dal12.pod01 | dal12 | 10.241.153.128/25 | single-tenant |
| `ci-vlan-1166-disconneted-dal12-dal12.pod01` | ci-vlan-1166-disconneted | 1166 | dal12.pod01 | dal12 | 10.241.153.128/25 | single-tenant |
| `ci-vlan-1169-dal12-dal12.pod01` | ci-vlan-1169 | 1169 | dal12.pod01 | dal12 | 10.241.157.0/25 | single-tenant |
| `ci-vlan-1169-disconneted-dal12-dal12.pod01` | ci-vlan-1169-disconneted | 1169 | dal12.pod01 | dal12 | 10.241.157.0/25 | single-tenant |
| `ci-vlan-1190-dal10-dal10.pod03` | ci-vlan-1190 | 1190 | dal10.pod03 | dal10 | 10.93.254.128/25 | single-tenant |
| `ci-vlan-1197-dal10-dal10.pod03` | ci-vlan-1197 | 1197 | dal10.pod03 | dal10 | 10.38.134.0/25 | single-tenant |
| `ci-vlan-1207-dal10-dal10.pod03` | ci-vlan-1207 | 1207 | dal10.pod03 | dal10 | 10.94.123.128/25 | single-tenant |
| `ci-vlan-1216-dal12-dal12.pod01` | ci-vlan-1216 | 1216 | dal12.pod01 | dal12 | 10.184.36.0/25 | single-tenant |
| `ci-vlan-1220-dal12-dal12.pod01` | ci-vlan-1220 | 1220 | dal12.pod01 | dal12 | 10.184.103.128/25 | single-tenant |
| `ci-vlan-1221-dal12-dal12.pod01` | ci-vlan-1221 | 1221 | dal12.pod01 | dal12 | 10.184.6.128/25 | single-tenant |
| `ci-vlan-1225-dal10-dal10.pod03` | ci-vlan-1225 | 1225 | dal10.pod03 | dal10 | 10.38.83.128/25 | single-tenant |
| `ci-vlan-1227-dal10-dal10.pod03` | ci-vlan-1227 | 1227 | dal10.pod03 | dal10 | 10.38.252.0/25 | single-tenant |
| `ci-vlan-1229-dal10-dal10.pod03` | ci-vlan-1229 | 1229 | dal10.pod03 | dal10 | 10.94.182.0/25 | single-tenant |
| `ci-vlan-1232-dal10-dal10.pod03` | ci-vlan-1232 | 1232 | dal10.pod03 | dal10 | 10.38.192.128/25 | single-tenant |
| `ci-vlan-1233-dal10-dal10.pod03` | ci-vlan-1233 | 1233 | dal10.pod03 | dal10 | 10.38.121.0/25 | single-tenant |
| `ci-vlan-1234-dal10-dal10.pod03` | ci-vlan-1234 | 1234 | dal10.pod03 | dal10 | 10.94.146.128/25 | single-tenant |
| `ci-vlan-1235-dal10-dal10.pod03` | ci-vlan-1235 | 1235 | dal10.pod03 | dal10 | 10.38.221.128/25 | single-tenant |
| `ci-vlan-1237-dal10-dal10.pod03` | ci-vlan-1237 | 1237 | dal10.pod03 | dal10 | 10.38.247.0/25 | single-tenant |
| `ci-vlan-1238-dal10-dal10.pod03` | ci-vlan-1238 | 1238 | dal10.pod03 | dal10 | 10.38.114.128/25 | single-tenant |
| `ci-vlan-1240-dal10-dal10.pod03` | ci-vlan-1240 | 1240 | dal10.pod03 | dal10 | 10.38.202.0/25 | single-tenant |
| `ci-vlan-1243-dal10-dal10.pod03` | ci-vlan-1243 | 1243 | dal10.pod03 | dal10 | 10.38.204.128/25 | single-tenant |
| `ci-vlan-1246-dal10-dal10.pod03` | ci-vlan-1246 | 1246 | dal10.pod03 | dal10 | 10.94.72.128/25 | single-tenant |
| `ci-vlan-1249-dal10-dal10.pod03` | ci-vlan-1249 | 1249 | dal10.pod03 | dal10 | 10.38.110.0/25 | single-tenant |
| `ci-vlan-1254-dal10-dal10.pod03` | ci-vlan-1254 | 1254 | dal10.pod03 | dal10 | 10.5.183.0/25 | single-tenant |
| `ci-vlan-1255-dal10-dal10.pod03` | ci-vlan-1255 | 1255 | dal10.pod03 | dal10 | 10.38.220.0/25 | single-tenant |
| `ci-vlan-1259-dal10-dal10.pod03` | ci-vlan-1259 | 1259 | dal10.pod03 | dal10 | 10.23.6.0/24 | single-tenant |
| `ci-vlan-1260-dal10-dal10.pod03` | ci-vlan-1260 | 1260 | dal10.pod03 | dal10 | 10.93.99.128/25 | single-tenant |
| `ci-vlan-1271-dal10-dal10.pod03` | ci-vlan-1271 | 1271 | dal10.pod03 | dal10 | 10.94.100.0/25 | single-tenant |
| `ci-vlan-1272-dal10-dal10.pod03` | ci-vlan-1272 | 1272 | dal10.pod03 | dal10 | 10.94.173.0/25 | single-tenant |
| `ci-vlan-1274-dal10-dal10.pod03` | ci-vlan-1274 | 1274 | dal10.pod03 | dal10 | 10.93.165.0/25 | single-tenant |
| `ci-vlan-1279-dal10-dal10.pod03` | ci-vlan-1279 | 1279 | dal10.pod03 | dal10 | 10.94.31.128/25 | single-tenant |
| `ci-vlan-1284-dal10-dal10.pod03` | ci-vlan-1284 | 1284 | dal10.pod03 | dal10 | 10.38.84.0/25 | nested-multi-tenant |
| `ci-vlan-1284-dal10-dal10.pod03-1` | ci-vlan-1284-1 | 1284 | dal10.pod03 | dal10 | 10.38.84.0/25 | multi-tenant |
| `ci-vlan-1284-dal10-dal10.pod03-2` | ci-vlan-1284-2 | 1284 | dal10.pod03 | dal10 | 10.38.84.0/25 | multi-tenant |
| `ci-vlan-1284-dal10-dal10.pod03-3` | ci-vlan-1284-3 | 1284 | dal10.pod03 | dal10 | 10.38.84.0/25 | multi-tenant |
| `ci-vlan-1284-dal10-dal10.pod03-4` | ci-vlan-1284-4 | 1284 | dal10.pod03 | dal10 | 10.38.84.0/25 | multi-tenant |
| `ci-vlan-1287-dal10-dal10.pod03` | ci-vlan-1287 | 1287 | dal10.pod03 | dal10 | 10.38.201.128/25 | single-tenant |
| `ci-vlan-1289-dal10-dal10.pod03` | ci-vlan-1289 | 1289 | dal10.pod03 | dal10 | 10.38.153.0/25 | single-tenant |
| `ci-vlan-1296-dal10-dal10.pod03` | ci-vlan-1296 | 1296 | dal10.pod03 | dal10 | 10.94.169.0/25 | single-tenant |
| `ci-vlan-1298-dal10-dal10.pod03` | ci-vlan-1298 | 1298 | dal10.pod03 | dal10 | 10.5.172.0/25 | single-tenant |
| `ci-vlan-1300-dal10-dal10.pod03` | ci-vlan-1300 | 1300 | dal10.pod03 | dal10 | 10.94.27.0/25 | single-tenant |
| `ci-vlan-1302-dal10-dal10.pod03` | ci-vlan-1302 | 1302 | dal10.pod03 | dal10 | 10.94.196.0/25 | single-tenant |
| `ci-vlan-832-dal10-dal10.pod03` | ci-vlan-832 | 832 | dal10.pod03 | dal10 | 10.93.68.0/25 | single-tenant |
| `ci-vlan-847-1-dal10-dal10.pod03` | ci-vlan-847 | 847 | dal10.pod03 | dal10 | 10.93.122.0/25 | single-tenant |
| `ci-vlan-852-1-dal10-dal10.pod03` | ci-vlan-852 | 852 | dal10.pod03 | dal10 | 10.95.248.0/25 | single-tenant |
| `ci-vlan-871-dal12-dal12.pod01` | ci-vlan-871 | 871 | dal12.pod01 | dal12 | 10.241.31.128/25 | single-tenant |
| `ci-vlan-879-1-dal10-dal10.pod03` | ci-vlan-879 | 879 | dal10.pod03 | dal10 | 10.95.242.0/25 | single-tenant |
| `ci-vlan-893-2-dal10-dal10.pod03` | ci-vlan-893 | 893 | dal10.pod03 | dal10 | 10.95.108.0/25 | single-tenant |
| `ci-vlan-894-1-dal10-dal10.pod03` | ci-vlan-894 | 894 | dal10.pod03 | dal10 | 10.95.168.128/25 | single-tenant |
| `ci-vlan-896-1-dal10-dal10.pod03` | ci-vlan-896 | 896 | dal10.pod03 | dal10 | 10.93.93.128/25 | nested-multi-tenant |
| `ci-vlan-910-1-dal10-dal10.pod03` | ci-vlan-910 | 910 | dal10.pod03 | dal10 | 10.95.160.0/25 | mutli-tenant |
| `ci-vlan-910-1-dal10-dal10.pod03-multi-1` | ci-vlan-910-1 | 910 | dal10.pod03 | dal10 | 10.95.160.0/25 | multi-tenant |
| `ci-vlan-910-1-dal10-dal10.pod03-multi-2` | ci-vlan-910-2 | 910 | dal10.pod03 | dal10 | 10.95.160.0/25 | multi-tenant |
| `ci-vlan-910-1-dal10-dal10.pod03-multi-3` | ci-vlan-910-3 | 910 | dal10.pod03 | dal10 | 10.95.160.0/25 | multi-tenant |
| `ci-vlan-910-1-dal10-dal10.pod03-multi-4` | ci-vlan-910-4 | 910 | dal10.pod03 | dal10 | 10.95.160.0/25 | multi-tenant |
| `ci-vlan-910-1-dal10-dal10.pod03-multi-5` | ci-vlan-910-5 | 910 | dal10.pod03 | dal10 | 10.95.160.0/25 | multi-tenant |
| `ci-vlan-910-1-dal10-dal10.pod03-multi-6` | ci-vlan-910-6 | 910 | dal10.pod03 | dal10 | 10.95.160.0/25 | multi-tenant |
| `ci-vlan-910-1-dal10-dal10.pod03-multi-7` | ci-vlan-910-7 | 910 | dal10.pod03 | dal10 | 10.95.160.0/25 | multi-tenant |
| `ci-vlan-918-2-dal10-dal10.pod03` | ci-vlan-918 | 918 | dal10.pod03 | dal10 | 10.93.157.0/25 | mutli-tenant |
| `ci-vlan-918-2-dal10-dal10.pod03-multi-1` | ci-vlan-918-1 | 918 | dal10.pod03 | dal10 | 10.93.157.0/25 | multi-tenant |
| `ci-vlan-918-2-dal10-dal10.pod03-multi-2` | ci-vlan-918-2 | 918 | dal10.pod03 | dal10 | 10.93.157.0/25 | multi-tenant |
| `ci-vlan-918-2-dal10-dal10.pod03-multi-3` | ci-vlan-918-3 | 918 | dal10.pod03 | dal10 | 10.93.157.0/25 | multi-tenant |
| `ci-vlan-918-2-dal10-dal10.pod03-multi-4` | ci-vlan-918-4 | 918 | dal10.pod03 | dal10 | 10.93.157.0/25 | multi-tenant |
| `ci-vlan-918-2-dal10-dal10.pod03-multi-5` | ci-vlan-918-5 | 918 | dal10.pod03 | dal10 | 10.93.157.0/25 | multi-tenant |
| `ci-vlan-918-2-dal10-dal10.pod03-multi-6` | ci-vlan-918-6 | 918 | dal10.pod03 | dal10 | 10.93.157.0/25 | multi-tenant |
| `ci-vlan-918-2-dal10-dal10.pod03-multi-7` | ci-vlan-918-7 | 918 | dal10.pod03 | dal10 | 10.93.157.0/25 | multi-tenant |
| `ci-vlan-922-dal12-dal12.pod01` | ci-vlan-922 | 922 | dal12.pod01 | dal12 | 10.184.15.128/25 | single-tenant |
| `ci-vlan-923-1-dal10-dal10.pod03` | ci-vlan-923 | 923 | dal10.pod03 | dal10 | 10.38.167.128/25 | single-tenant |
| `ci-vlan-937-dal12-dal12.pod01` | ci-vlan-937 | 937 | dal12.pod01 | dal12 | 10.241.72.0/24 | public-ipv6 |
| `ci-vlan-938-1-dal10-dal10.pod03` | ci-vlan-938 | 938 | dal10.pod03 | dal10 | 10.93.251.0/25 | mutli-tenant |
| `ci-vlan-938-1-dal10-dal10.pod03-multi-1` | ci-vlan-938-1 | 938 | dal10.pod03 | dal10 | 10.93.251.0/25 | multi-tenant |
| `ci-vlan-938-1-dal10-dal10.pod03-multi-2` | ci-vlan-938-2 | 938 | dal10.pod03 | dal10 | 10.93.251.0/25 | multi-tenant |
| `ci-vlan-938-1-dal10-dal10.pod03-multi-3` | ci-vlan-938-3 | 938 | dal10.pod03 | dal10 | 10.93.251.0/25 | multi-tenant |
| `ci-vlan-938-1-dal10-dal10.pod03-multi-4` | ci-vlan-938-4 | 938 | dal10.pod03 | dal10 | 10.93.251.0/25 | multi-tenant |
| `ci-vlan-938-1-dal10-dal10.pod03-multi-5` | ci-vlan-938-5 | 938 | dal10.pod03 | dal10 | 10.93.251.0/25 | multi-tenant |
| `ci-vlan-938-1-dal10-dal10.pod03-multi-6` | ci-vlan-938-6 | 938 | dal10.pod03 | dal10 | 10.93.251.0/25 | multi-tenant |
| `ci-vlan-938-1-dal10-dal10.pod03-multi-7` | ci-vlan-938-7 | 938 | dal10.pod03 | dal10 | 10.93.251.0/25 | multi-tenant |
| `ci-vlan-946-1-dal10-dal10.pod03` | ci-vlan-946 | 946 | dal10.pod03 | dal10 | 10.38.133.128/25 | single-tenant |
| `ci-vlan-956-dal10-dal10.pod03` | ci-vlan-956 | 956 | dal10.pod03 | dal10 | 10.93.134.0/25 | single-tenant |
| `ci-vlan-958-1-dal10-dal10.pod03` | ci-vlan-958 | 958 | dal10.pod03 | dal10 | 10.93.152.0/25 | mutli-tenant |
| `ci-vlan-958-1-dal10-dal10.pod03-multi-1` | ci-vlan-958-1 | 958 | dal10.pod03 | dal10 | 10.93.152.0/25 | multi-tenant |
| `ci-vlan-958-1-dal10-dal10.pod03-multi-2` | ci-vlan-958-2 | 958 | dal10.pod03 | dal10 | 10.93.152.0/25 | multi-tenant |
| `ci-vlan-958-1-dal10-dal10.pod03-multi-3` | ci-vlan-958-3 | 958 | dal10.pod03 | dal10 | 10.93.152.0/25 | multi-tenant |
| `ci-vlan-958-1-dal10-dal10.pod03-multi-4` | ci-vlan-958-4 | 958 | dal10.pod03 | dal10 | 10.93.152.0/25 | multi-tenant |
| `ci-vlan-958-1-dal10-dal10.pod03-multi-5` | ci-vlan-958-5 | 958 | dal10.pod03 | dal10 | 10.93.152.0/25 | multi-tenant |
| `ci-vlan-958-1-dal10-dal10.pod03-multi-6` | ci-vlan-958-6 | 958 | dal10.pod03 | dal10 | 10.93.152.0/25 | multi-tenant |
| `ci-vlan-958-1-dal10-dal10.pod03-multi-7` | ci-vlan-958-7 | 958 | dal10.pod03 | dal10 | 10.93.152.0/25 | multi-tenant |
| `ci-vlan-976-1-dal10-dal10.pod03` | ci-vlan-976 | 976 | dal10.pod03 | dal10 | 10.93.117.128/25 | single-tenant |
| `ci-vlan-981-1-dal10-dal10.pod03` | ci-vlan-981 | 981 | dal10.pod03 | dal10 | 10.93.60.128/25 | nested-multi-tenant |
| `ci-vlan-981-1-dal10-dal10.pod03-2` | ci-vlan-981 | 981 | dal10.pod03 | dal10 | 10.93.60.128/25 | multi-tenant |
| `ci-vlan-981-1-dal10-dal10.pod03-3` | ci-vlan-981 | 981 | dal10.pod03 | dal10 | 10.93.60.128/25 | multi-tenant |
| `ci-vlan-981-1-dal10-dal10.pod03-4` | ci-vlan-981 | 981 | dal10.pod03 | dal10 | 10.93.60.128/25 | multi-tenant |
| `ci-vlan-988-dal12-dal12.pod01` | ci-vlan-988 | 988 | dal12.pod01 | dal12 | 10.184.115.0/25 | single-tenant |
| `ci-vlan-990-dal12-dal12.pod01` | ci-vlan-990 | 990 | dal12.pod01 | dal12 | 10.241.112.0/25 | single-tenant |
| `ci-vlan-990-disconneted-dal12-dal12.pod01` | ci-vlan-990-disconneted | 990 | dal12.pod01 | dal12 | 10.241.112.0/25 | single-tenant |
| `ci-vlan-991-dal12-dal12.pod01` | ci-vlan-991 | 991 | dal12.pod01 | dal12 | 10.241.99.128/25 | single-tenant |
| `nested-vcenter-7-vmnetwork` | ci-vlan-981-testing-nested7 | 981 | dal10.pod03 | dal10 | 10.93.60.128/25 | multi-tenant |
| `nested-vcenter-7-vmnetwork-1` | ci-vlan-981-testing-nested7-1 | 981 | dal10.pod03 | dal10 | 10.93.60.128/25 | multi-tenant |
| `nested-vcenter-7-vmnetwork-2` | ci-vlan-981-testing-nested7-2 | 981 | dal10.pod03 | dal10 | 10.93.60.128/25 | multi-tenant |

## Refreshing this document

From a machine with cluster access:

```sh
NS=vsphere-infra-helpers
oc config current-context
oc get pools.vspherecapacitymanager.splat.io -n "$NS" -o wide
oc get networks.vspherecapacitymanager.splat.io -n "$NS"
```

To rebuild the tables, re-run the `jq` snippets against `oc get ... -o json` and update the markdown manually. Omit sensitive annotations (for example Vault paths) if you paste full objects.

