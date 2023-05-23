deploy: containers
	./deploy.sh

containers:
	cd services/service1 \
		&& docker build -f Dockerfile            -t service1:0.2 . \
		&& docker build -f migrations/Dockerfile -t service1-migrations:0.2 .

	cd services/service2 \
		&& docker build -f Dockerfile            -t service2:0.2 . \
		&& docker build -f migrations/Dockerfile -t service2-migrations:0.2 .

	cd client && docker build -f Dockerfile -t client:0.2 .
