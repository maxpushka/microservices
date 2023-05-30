#!/bin/bash

# Check if Minikube is running
if ! minikube status &> /dev/null; then
    echo "Minikube is not running. Starting Minikube..."

    # Start Minikube
    if ! minikube start; then
        echo "Failed to start Minikube."
        exit 1
    fi
else
    echo "Minikube is already running."
fi

# Clean up previous deployment
echo "Uninstalling previous deployment"
helm uninstall local
istioctl uninstall --purge -y
kubectl delete namespaces istio-system

# Build containers
cd ./services/service1
docker build -f Dockerfile            -t service1:0.5            .
docker build -f migrations/Dockerfile -t service1-migrations:0.5 .
cd -

cd ./services/service2
docker build -f Dockerfile            -t service2:0.5            .
docker build -f migrations/Dockerfile -t service2-migrations:0.5 .
cd -

echo "Building logger"
cd ./services/logger
docker build -f Dockerfile -t logger:0.5 .
cd -

echo "Building client"
cd ./client
docker build -f Dockerfile -t client:0.5 .
cd -

# Push containers to Minikube's registry
eval $(minikube -p minikube docker-env)

echo "Pushing service1 to Minikube..."
minikube image load service1:0.5
minikube image load service1-migrations:0.5
echo "Pushing service2 to Minikube..."
minikube image load service2:0.5
minikube image load service2-migrations:0.5
echo "Pushing logger to Minikube..."
minikube image load logger:0.5
echo "Pushing client to Minikube..."
minikube image load client:0.5

# Deploy Istio
echo "Deploying Istio"
kubectl create namespace istio-system
helm install istio-base istio/base -n istio-system
helm install istiod istio/istiod -n istio-system --wait
kubectl label namespace default istio-injection=enabled

# Deploy services to cluster
echo "Deploying services to cluster"
helm install local helm/v1
kubectl proxy
