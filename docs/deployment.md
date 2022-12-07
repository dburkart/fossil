# Deployment

- [Deployment](#deployment)
  - [Containerized](#containerized)
    - [Docker](#docker)
    - [Kubernetes](#kubernetes)
      - [Deploy](#deploy)
      - [Config](#config)
  - [Binary Distributions](#binary-distributions)
    - [Build](#build)
    - [Run](#run)


Fossil recommendations
- Memory
  - Usage: 500Mi
- CPU
  - Usage: 500m



## Containerized

### Docker

Build
```shell
docker build -t fossil .
```
or Pull
```shell
docker pull gideonw/fossil:latest
```
and Run
```shell
docker run -it -p8001:8001 -p2112:2112 gideonw/fossil:latest server
```

### Kubernetes
Apply the following resources from the `deploy/kubernetes` directory within the repo.

Objects:
- Deployment
- Service
- Ingress

#### Deploy
Pull and clone the repository in-order to use the deploy/kubernetes example deployment.
```shell
kubectl create namespace fossil
```
```shell
kubectl config set-context --current --namespace=fossil
```
```shell
kubectl apply -f ./deploy/kubernetes/
```

#### Config
Modify the `deploy/kubernetes/config.yaml` file to update the configuration for the fossil deployment.

## Binary Distributions

### Build

### Run