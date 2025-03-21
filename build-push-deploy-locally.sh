#!/bin/bash

function wait_for_resource() {
  local sleep=3
  local resource="$1"
  local name="$2"
  local namespace="${3:-}"
  local jsonpath="${4:-}"
  local grep="${5:-}"
  args=("$resource" "$name")
  if [ "$namespace" == "all" ]; then
    args+=("--all-namespaces")
  elif [ -n "$namespace" ]; then
    args+=("-n" "$namespace")
  fi
  if [ -n "$jsonpath" ]; then
    args+=("-o" "jsonpath=$jsonpath")
  fi
  echo "Polling for $resource $name in $namespace"
  if [ -z "$grep" ]; then
    until kubectl get "${args[@]}"; do echo "..." && sleep "$sleep"; done
  else
    until kubectl get "${args[@]}" | grep -q -- "$grep"; do echo "..." && sleep "$sleep"; done
  fi
}

TAG=$1

echo "Running: make generate"
make generate

echo ""
echo "Running: build -t local-registry:5000/mapserver-operator:$TAG --build-context repos=./.. ."
docker build -t "local-registry:5000/mapserver-operator:$TAG" --build-context repos=./.. .

echo ""
echo "Running: push local-registry:5000/mapserver-operator:$TAG"
docker push "local-registry:5000/mapserver-operator:$TAG"

if [[ $(kubectl get pod -l app=webhook -n cert-manager | grep "cert-manager") ]]; then
  echo "Cert-manager already installed"
else
  echo ""
  echo "Installing cert-manager"
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.0/cert-manager.yaml
fi

echo "Waiting for cert-manager"
while [[ $(kubectl get pod -l app=webhook -n cert-manager -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do
  sleep 1
done
echo "Cert-manager ready"

echo ""
echo "Running: make install"
make install

echo ""
echo "Running: deploy IMG=local-registry:5000/mapserver-operator:$TAG"
make deploy "IMG=local-registry:5000/mapserver-operator:$TAG"