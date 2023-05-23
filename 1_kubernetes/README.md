# Lab 1: Kubernetes

## How to start

To run the project, run `make deploy` in terminal

``` sh
$ curl http://localhost:8001/api/v1/namespaces/default/services/service1-service/proxy/
Hello, world!

$ curl http://localhost:8001/api/v1/namespaces/default/services/service2-service/proxy/?name=test
Hello, test!
```

