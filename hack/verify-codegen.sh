#!/bin/sh

if [ "$IS_CONTAINER" != "" ]; then
  set -xe
  go generate ./pkg/apis/vsphere-capacity-manager.splat-team.io/install.go
  set +ex
  # git diff --exit-code
else
  podman run --rm \
    --env IS_CONTAINER=TRUE \
    --volume "${PWD}:/go/src/github.com/openshift/vsphere-capacity-manager:z" \
    --workdir /go/src/github.com/openshift/vsphere-capacity-manager \
    docker.io/golang:1.18 \
    ./hack/verify-codegen.sh "${@}"
fi
