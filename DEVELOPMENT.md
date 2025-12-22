## Local testing

- Start an empty cluster using `k8s-clusters/local-test/empty-cluster.sh`
- Build and push the controller to the cluster using `build-and-push-locally.sh <controller-version>`
- Deploy a service to the cluster, for example (running from `k8s-clusters/local-test`): `OWNER=kadaster TECHNICAL_NAME=ad docker-compose -f ./docker-compose.yaml -f ./bundle-pollers/docker-compose.services.yaml up kustomize-init`
