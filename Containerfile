FROM golang:1.22 AS builder
WORKDIR /go/src/github.com/openshift-splat-team/vsphere-capacity-manager
COPY . .
ENV GO_PACKAGE github.com/openshift-splat-team/vsphere-capacity-manager
RUN NO_DOCKER=1 make build

# FROM registry.ci.openshift.org/openshift/origin-v4.0:base
# FROM registry.ci.openshift.org/ocp/4.13:base
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.7-1107
COPY --from=builder /go/src/github.com/openshift-splat-team/vsphere-capacity-manager/bin/vsphere-capacity-manager /usr/bin/vsphere-capacity-manager
ENTRYPOINT ["/usr/bin/vsphere-capacity-manager"]
LABEL io.openshift.release.operator=true
