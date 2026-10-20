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
    * [Dependency management 🔧](#dependency-management)
        + [Dependency lock](#dependency-lock)
        + [Dependency verification](#dependency-verification)
    * [Contributors 👥](#contributors-)
        + [Maintainers](#maintainers)

## Scope
This microservice is responsible for help desk api to be exposed in order to perform queries against registered wallets, their statuses and so on

---

## Api Documentation 📖

See the [OpenAPI 3 here.](https://editor.swagger.io/?url=https://raw.githubusercontent.com/pagopa/pagopa-payment-wallet-helpdesk-service/main/api-spec/api.yaml)

---

## Technology Stack

- [Go](https://go.dev/learn/) 1.27.3 (also used by the Docker builder)
- [GIN](https://go.dev/doc/tutorial/web-service-gin)
- Alpine 3.24.2 runtime image, pinned by digest
- MongoDB Go driver 1.17.10. The v1 driver is deprecated; migration to v2 is deferred
  pending compatibility verification with the target Azure Cosmos DB MongoDB API.
  Gin also pulls in the v2 BSON package transitively; application database access
  continues to use the v1 driver.

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
- Go 1.27.3 or newer, with the same version recommended locally and in Docker
- make

### Run the project

```shell
$ go build -o ./build/payment-wallet-helpdesk-service ./cmd/app
$ ./build/payment-wallet-helpdesk-service
```


### Testing 🧪

#### Unit testing

```shell
make test
```

Runs all Go tests with the race detector. The MongoDB client test checks client
creation only; it does not verify a connection or execute database queries.

#### Integration testing

Integration tests against MongoDB and the target Azure Cosmos DB instance are
not yet automated. Verify database connectivity and wallet queries in the target
environment before releasing.

The application removes the OpenAPI `servers` list from its in-memory validator
specification, so documented deployment URLs do not restrict local or Kubernetes
probe requests. The published specification is unchanged; request parameter,
body and security validation remain enabled.

After `make run`, call the local endpoint with:

```shell
curl -i http://localhost:8080/info
```

`GET /info` is public (`security: []` in the OpenAPI operation) and returns HTTP
200 with the service version, without an `x-api-key` header. This also allows
the Kubernetes liveness and readiness probes to reach the endpoint.

Wallet endpoints retain the global `ApiKeyAuth` requirement. The application does
not yet configure an authentication function for the validator, so wallet requests
return HTTP 400 with a security validation error, even if an `x-api-key` header is
supplied. Authentication must be configured before protected endpoints can be used.

For wallet queries, replace the router placeholder with a real user ID, for
example `/helpdesk/wallets/12345`, not `/helpdesk/wallets/:userId`.

#### Performance testing

Performance tests are not yet automated.

## Dependency management

### Dependency lock

Commit both `go.mod` and `go.sum` after dependency updates. Docker copies both
files before downloading modules. Base images are pinned by version and digest.

The OpenAPI generator is pinned in `go.mod`; regenerate bindings without changing
the API specification with:

```shell
go generate ./cmd/app/api
```

Keep the generator's YAML overlay dependencies at the compatible versions in
`go.mod` (`speakeasy-api/openapi` 1.24.0, `speakeasy-api/jsonpath` 0.6.3 and
`dprotaso/go-yit` at the pinned 2022 revision). Updating these transitives
indiscriminately currently introduces incompatible YAML node types and breaks
generation.

Staticcheck 0.8.2, govulncheck 1.9.0 and Air 1.67.4 are pinned in the Makefile.
When updating Go, also check tool compatibility and update the image digest.

### Dependency verification

```shell
make audit
make build
```

`make audit` runs tests, checks module tidiness without rewriting files, verifies
module checksums and formatting, and runs `go vet`, Staticcheck and govulncheck.
Staticcheck retains all configured checks for handwritten application code; the
generated API package excludes ST1005 for generator-owned error messages.

The vulnerability scan uses the current Go vulnerability database. After this
upgrade it reports no reachable vulnerabilities and no vulnerable imported
packages. It still reports GO-2026-5932 at module level for the unused
`golang.org/x/crypto/openpgp` package, which has no fixed version.

## Contributors 👥

Made with ❤️ by PagoPA S.p.A.

### Maintainers

See `CODEOWNERS` file
