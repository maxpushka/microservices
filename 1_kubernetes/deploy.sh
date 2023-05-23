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

eval $(minikube -p minikube docker-env)

# Push containers to Minikube's registry
minikube image load service1:0.1
minikube image load service2:0.1
minikube image load client:0.1

# Deploy to cluster
kubectl apply -f k8s/client -f k8s/service1 -f k8s/service2
kubectl proxy
