deploy: containers
	./deploy.sh

containers:
	cd services/service1 && docker build -f Dockerfile -t service1:0.1 .
	cd services/service2 && docker build -f Dockerfile -t service2:0.1 .
	cd client && docker build -f Dockerfile -t client:0.1 .
