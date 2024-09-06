# pagoPA Payment Wallet help desk service

- [Scope](#scope)
    * [Api Documentation 📖](#api-documentation-)
    * [Technology Stack](#technology-stack)
    * [Start Project Locally 🚀](#start-project-locally-)
        + [Prerequisites](#prerequisites)
        + [Run docker container](#run-docker-container)
    * [Develop Locally 💻](#develop-locally-)
        + [Prerequisites](#prerequisites-1)
        + [Run the project](#run-the-project)
        + [Install ecommerce commons library](#install-ecommerce-commons-library-locally)
        + [Testing 🧪](#testing-)
            - [Unit testing](#unit-testing)
            - [Integration testing](#integration-testing)
            - [Performance testing](#performance-testing)
    * [Dependency management 🔧](#dependency-management-)
        + [Dependency lock](#dependency-lock)
        + [Dependency verification](#dependency-verification)
    * [Contributors 👥](#contributors-)
        + [Maintainers](#maintainers)

## Scope
This microservice is responsible for help desk api to be exposed in order to perform queries against registered wallets, their statuses and so on

---

## Api Documentation 📖

See the [OpenAPI 3 here.](https://editor.swagger.io/?url=https://raw.githubusercontent.com/pagopa/pagopa-payment-wallet-helpdesk-service/main/api-spec/openapi.yaml)

---

## Technology Stack

- [GO](https://go.dev/learn/)
- [GIN](https://go.dev/doc/tutorial/web-service-gin)

---

## Start Project Locally 🚀

### Prerequisites

- docker

### Populate the environment

The microservice needs a valid `.env` file in order to be run.

If you want to start the application without too much hassle, you can just copy `.env.example` with

```shell
$ cp .env.example .env
```

to get a good default configuration.

If you want to customize the application environment, reference this table:

| Variable name | Description | type | default |
|---------------|-------------|------|---------|
 |               |             |      |         | 


### Run docker container

```shell
$ docker compose up --build
```

---

## Develop Locally 💻

There is a Makefile that makes develop locally easy.
You can build and run the application with the following commmands

```shell
$ make build
$ make run
```
This application require env variables to be set
There is support for live reload also using the be set.
This can be done with the following command

```shell
$ export $(cat .env | xargs)
```

```shell
$ make run/live
```

or with a single command as

```shell
$ export $(cat .env | xargs) && $ make run/live
```

See make help for all supported commands

### Prerequisites

- git
- go

### Run the project

```shell
$ go build -o ./build/payment-wallet-helpdesk-service
$ ./build/payment-wallet-helpdesk-service
```


### Testing 🧪

#### Unit testing



#### Integration testing

TODO

#### Performance testing

## Contributors 👥

Made with ❤️ by PagoPA S.p.A.

### Maintainers

See `CODEOWNERS` file
