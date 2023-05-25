#!/bin/bash

# Build containers
cd ./services/service1
docker build -f Dockerfile            -t service1:0.2            .
docker build -f migrations/Dockerfile -t service1-migrations:0.2 .
cd -

cd ./services/service2
docker build -f Dockerfile            -t service2:0.2            .
docker build -f migrations/Dockerfile -t service2-migrations:0.2 .
cd -

cd ./client
docker build -f Dockerfile -t client:0.2 .
cd -

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
echo "Pushing service1 to Minikube..."
minikube image load service1:0.2
minikube image load service1-migrations:0.2
echo "Pushing service2 to Minikube..."
minikube image load service2:0.2
minikube image load service2-migrations:0.2
echo "Pushing client to Minikube..."
minikube image load client:0.2

# Deploy to cluster
kubectl apply -f k8s/client -f k8s/postgres -f k8s/service1 -f k8s/service2
kubectl proxy
