#!/bin/bash

TAG=$1

echo "Running: make generate"
make generate

echo ""
echo "Running: build -t local-registry:5000/wfs-wms-operator:$TAG --build-context repos=./.. ."
docker build -t "local-registry:5000/wfs-wms-operator:$TAG" --build-context repos=./.. .

echo ""
echo "Running: push local-registry:5000/wfs-wms-operator:$TAG"
docker push "local-registry:5000/wfs-wms-operator:$TAG"

echo ""
echo "Installing cert-manager"
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.0/cert-manager.yaml

echo ""
echo "Running: make install"
make install

echo ""
echo "Running: deploy IMG=local-registry:5000/wfs-wms-operator:$TAG"
make deploy "IMG=local-registry:5000/wfs-wms-operator:$TAG"