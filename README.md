# poc-kubernetes-batch

PoC to create a new k8s Job using the Golang API, making use of Init Containers.

## Setup

```
# Kubernetes
brew cask install minikube
brew install docker-machine-driver-xhyve
brew install kubectl
minikube start --vm-driver=xhyve
eval $(minikube docker-env)
kubectl create -f ./namespace.yaml

# Dependency Management
brew install glide
glide up -v // AKA: glide update --strip-vendor
```

## Usage

```
eval $(minikube docker-env)

go run main.go
```

## Improvements

* Switch from Glide to Dep [when Kubernetes client-go supports it](https://github.com/kubernetes/client-go/blob/master/INSTALL.md#dep-not-supported-yet)

## References, etc

* https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
* https://v1-8.docs.kubernetes.io/docs/api-reference/v1.8/
* https://github.com/kubernetes/client-go/tree/master/examples
* https://stackoverflow.com/questions/32554893/how-can-i-create-a-simple-client-app-with-the-kubernetes-go-library
