
# User Service

This is a simple user service written in Go.

## Project Organization

*   `cmd/server/main.go`: This is the main entry point of the application. It initializes the config, metrics, services, and handlers, and then starts the HTTP server.

*   `deployments`: This directory contains all the files related to deploying the application. It's further subdivided into `docker`, `k8s`, and `monitoring`.
    *   `docker`: Contains the `Dockerfile` and `docker-compose.yml` files for building and running the application with Docker.
    *   `k8s`: Intended for Kubernetes deployment files.
    *   `monitoring`: Contains the monitoring stack, including Prometheus and Grafana configurations.

*   `docs`: This directory is for documentation.

*   `internal`: This is the heart of the application, containing all the core business logic. It's subdivided into several packages:
    *   `config`: Handles loading configuration from environment variables.
    *   `handlers`: Contains the HTTP handlers that respond to incoming requests.
    *   `metrics`: Sets up and manages the Prometheus metrics.
    *   `middleware`: Contains the HTTP middleware, such as logging, metrics, and rate limiting.
    *   `models`: Defines the data structures used in the application, such as the `User` struct.
    *   `services`: Contains the business logic of the application, such as the `UserService`.

*   `scripts`: This directory contains various scripts for building, testing, and running the application.

*   `test/integration`: This directory contains the integration tests for the application.

This structure separates the application into logical components, making it easier to understand, maintain, and extend. The `internal` directory is not meant to be imported by other applications, which helps to enforce encapsulation and prevent tight coupling. The `deployments` directory keeps all the deployment-related files in one place, making it easy to deploy the application to different environments.

## Usage

To build the application, run:

```
make build
```

To run the application, run:

```
make run
```

To run the tests, run:

```
make test
```
