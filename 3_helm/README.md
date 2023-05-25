# Lab 3: Kubernetes

## How to start

To run the project, run `./deploy.sh` in terminal

## Available handlers

### Service 1

**Service 1** sets up an HTTP server with the following endpoints:

- `GET /users`: Retrieves all users from the database.
- `POST /users`: Creates a new user by sending a JSON payload containing the username and email.
- `GET /users/{id}`: Retrieves a specific user by ID.
- `PUT /users/{id}`: Updates a specific user by ID with a JSON payload containing the updated username and email.
- `DELETE /users/{id}`: Deletes a specific user by ID.

### Service 2

**Service 2** sets up an HTTP server with the following endpoints:

- `GET /products`: Retrieves all products from the database.
- `POST /products`: Creates a new product by sending a JSON payload containing the name and price.
- `GET /products/{id}`: Retrieves a specific product by ID.
- `PUT /products/{id}`: Updates a specific product by ID with a JSON payload containing the updated name and price.
- `DELETE /products/{id}`: Deletes a specific product by ID.


``` sh
$ curl http://localhost:8001/api/v1/namespaces/default/services/service2-service/proxy/products
Hello, world!
```

