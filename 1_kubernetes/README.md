# Lab 1: Kubernetes

## How to start

To run the project, run `make deploy` in terminal

## Available handlers

### Client

Returns HTML page with operational statuses of deployed services.

``` sh
$ curl http://localhost:8001/api/v1/namespaces/default/services/client-service/proxy/
<!DOCTYPE html>
<!-- . . . -->
```


### Service 1

A *hello world* handler.

``` sh
$ curl http://localhost:8001/api/v1/namespaces/default/services/service1-service/proxy/
Hello, world!
```

### Service 2

It's possible to send query param `name` to greet specific user:

``` sh
$ curl http://localhost:8001/api/v1/namespaces/default/services/service2-service/proxy/?name=test
Hello, test!
```

