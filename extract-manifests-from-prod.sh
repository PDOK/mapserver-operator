#!/bin/bash

SERVICE_TYPE=${1:-wfs}

ORIGINAL_KUBECONFIG=$(echo $KUBECONFIG)
export KUBECONFIG=/Users/jelledijkstra/.kube/aks_config_prod
kubectx aks-services-oostwoud
SERVICES=$(kubectl get $SERVICE_TYPE -n services)

MANIFESTS_DIR=prod-manifests/$SERVICE_TYPE
mkdir -p $MANIFESTS_DIR
rm "$MANIFESTS_DIR/"*.json >/dev/null 2>&1
rm "$MANIFESTS_DIR/"*.yaml >/dev/null 2>&1

python3 -m pip install pyyaml

REMOVE_KEYS=('.metadata.annotations."kubectl.kubernetes.io/last-applied-configuration"' ".status" ".metadata.creationTimestamp" ".metadata.generation" ".metadata.uid" ".metadata.resourceVersion" ".metadata.namespace")

IFS=$'\n'
LINENUM=-1
for SERVICE in $SERVICES; do
  LINENUM=$(expr $LINENUM + 1)

  if [[ $LINENUM -eq 0 ]]; then
    continue
  fi

  SERVICE=$(echo $SERVICE | awk '{print $1}')

  JSON="$MANIFESTS_DIR/$SERVICE.json"
  kubectl get $SERVICE_TYPE/$SERVICE -n services -o json > "$JSON"

  for KEY in "${REMOVE_KEYS[@]}"; do
    jq "del($KEY)" "$JSON" > "$JSON.tmp" && mv "$JSON.tmp" "$JSON"
  done

  YAML="$MANIFESTS_DIR/$SERVICE.yaml"
  cat "$JSON" | python3 -c 'import sys, yaml, json; print(yaml.dump(json.loads(sys.stdin.read())))' > "$YAML"
  rm "$JSON"

  # Replace column y with "y" - otherwise the admission controller thinks its a boolean
  sed 's/- y$/- "y"/g' "$YAML" > "$YAML.tmp" && mv "$YAML.tmp" "$YAML"
done

export KUBECONFIG=$ORIGINAL_KUBECONFIG