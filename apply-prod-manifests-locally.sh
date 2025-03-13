#!/bin/bash

KUBECTX=$(kubectx)
if [[ "$KUBECTX" != "default" ]]; then
  echo "You need to be connected with the local cluster."
  exit 1
fi

SERVICE_TYPE=${1:-wfs}

for MANIFEST in "./prod-manifests/$SERVICE_TYPE/"*.yaml; do
  kubectl apply -f $MANIFEST

  if [ $? -eq 0 ]; then
    kubectl delete -f $MANIFEST
  else
    break
  fi
done