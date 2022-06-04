[![Go Report Card](https://goreportcard.com/badge/kubevault.dev/secrets-store-reader)](https://goreportcard.com/report/kubevault.dev/secrets-store-reader)
[![Build Status](https://github.com/kubevault/secrets-store-reader/workflows/CI/badge.svg)](https://github.com/kubevault/secrets-store-reader/actions?workflow=CI)
[![Docker Pulls](https://img.shields.io/docker/pulls/kubevault/secrets-store-reader.svg)](https://hub.docker.com/r/kubevault/secrets-store-reader/)
[![Slack](https://shields.io/badge/Join_Slack-salck?color=4A154B&logo=slack)](https://slack.appscode.com)
[![Twitter](https://img.shields.io/twitter/follow/kubevault.svg?style=social&logo=twitter&label=Follow)](https://twitter.com/intent/follow?screen_name=KubeVault)

# secrets-store-reader

Today [Secrets Store CSI driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver) expects applications to mount any secrets coming from the various supported external drivers. `secrets-store-reader` is an [extended api server](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/) exposes these secrets as a "Secret" like resource.

## Deploy into a Kubernetes Cluster

You can deploy `secrets-store-reader` using Helm chart found [here](https://github.com/kubevault/installer/tree/master/charts/secrets-store-reader).

```console
helm repo add appscode https://charts.appscode.com/stable/
helm repo update

helm install secrets-store-reader appscode/secrets-store-reader
```
