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
    * [APIs](#apis)
        + [Contract ID deactivation DLQ search](#contract-id-deactivation-dlq-search)
        + [Reset wallet deactivation retry counter](#reset-wallet-deactivation-retry-counter)
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

to get the local database configuration. Before starting, set
`SECURITY_API_KEYS_PRIMARY` and `SECURITY_API_KEYS_SECONDARY` in `.env` to two
independently generated secret values. They are intentionally empty in
`.env.example`; there are no default API keys.

If you want to customize the application environment, reference this table:

| Variable name | Description | type | default |
|---------------|-------------|------|---------|
| `MONGO_DB_NAME` | Database shared by wallet, DLQ and event collections; set `payment-wallet` for the target wallet database | string | `wallet` |
| `PAYMENT_WALLET_CONTRACT_DEACTIVATION_DLQ_COLLECTION_NAME` | DLQ collection | string | `wallet-contractId-deactivation-dlq` |
| `PAYMENT_WALLET_LOG_EVENTS_COLLECTION_NAME` | Wallet audit event collection | string | `payment-wallets-log-events` |
| `SECURITY_API_KEYS_SECURED_PATHS` | Required comma-separated path prefixes protected by the API key middleware | string | No application default; `/helpdesk/` in example and Helm |
| `SECURITY_API_KEYS_PRIMARY` | Required primary API key | secret string | None |
| `SECURITY_API_KEYS_SECONDARY` | Required secondary API key, accepted alongside primary for rotation | secret string | None |

Configuration is loaded once at startup. Empty/blank keys, empty prefix lists,
empty list entries and invalid path prefixes prevent startup. Prefixes must
start with `/` and contain no whitespace, query string or fragment. Prefix list
whitespace is trimmed; API keys are compared exactly and never trimmed.

The Helm development values reference the Key Vault secrets
`payment-wallet-helpdesk-service-primary-api-key` and
`payment-wallet-helpdesk-service-secondary-api-key`. Provision these secrets before
deploying, or change the `envSecret` mappings to the actual secret names.
APIM must forward `x-api-key` with one of these service key values; an APIM
subscription key is not automatically a valid service key. Rotate one key at a
time, restarting the application to load updated values while the other key
remains usable.

### Run docker container

```shell
$ docker compose up --build
```

On a fresh MongoDB data directory, `docker/mongodb/mongo-init.js` seeds the
`wallet` database with wallets, three DLQ entries and their deactivation error
events, including retries and an older contract. It also creates the DLQ and
event lookup compound indexes. Use `MONGO_DB_NAME=wallet` for this sample data.
The initialization script does not run again on an already initialized data
directory; restarting an existing container does not update its data.

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
Reset service unit tests cover the 204/404/500 response mapping without MongoDB.

#### Integration testing

DLQ repository and HTTP integration tests can run against a local MongoDB instance
as described below. Integration tests against the target Azure Cosmos DB instance
are not yet automated. Verify database connectivity, indexes and wallet queries
in the target environment before releasing.

The application removes the OpenAPI `servers` list from its in-memory validator
specification, so documented deployment URLs do not restrict local or Kubernetes
probe requests. The published specification is unchanged; request parameter and
body validation remain enabled. A Gin middleware validates the API key before
the OpenAPI schema validator. APIM remains an additional authentication and
authorization layer.
All request validator errors use the `ProblemJson` shape (`status`, `title`,
`detail`), including wallet parameter errors and unknown operations. This replaces
the previous `{"error": "..."}` body for non-DLQ routes; HTTP statuses are unchanged.

After `make run`, call the local endpoint with:

```shell
curl -i http://localhost:8080/info
```

`GET /info` is public (`security: []` in the OpenAPI operation) and returns HTTP
200 with the service version, without an `x-api-key` header. This also allows
the Kubernetes liveness and readiness probes to reach the endpoint.

Wallet endpoints retain the global `ApiKeyAuth` requirement in the published
OpenAPI contract for APIM and API consumers. The API key middleware protects
every HTTP method whose decoded request path starts with one of
`SECURITY_API_KEYS_SECURED_PATHS` (case-sensitive prefix matching). Keep
`/helpdesk/` in that list to cover all wallet operations. Other prefixes can be
added as comma-separated values. `GET /info` is always public, even if a
configured prefix would otherwise match it.

Protected requests require exactly one non-empty `x-api-key` header matching
either the primary or secondary key. Missing, invalid, combined or duplicate
headers return HTTP 401 with `ProblemJson`, before schema validation or database
access. API keys are compared using constant-time comparisons of fixed-size
SHA-256 hashes, and neither received nor configured keys are logged.
Authentication failures are logged without header values.
The OpenAPI validator retains `openapi3filter.NoopAuthenticationFunc` because
the preceding middleware performs authentication; schema and parameter validation
still apply to authenticated requests.

Before deployment, verify that APIM policies enforce authentication and
authorization and that network and ingress restrictions prevent clients from
reaching wallet endpoints directly, bypassing APIM. Direct local access now
requires a service API key as well; database connectivity is still needed for
wallet queries.

In Postman, set the `api-key` variable locally to either configured service key.
Wallet requests inherit collection-level `x-api-key` authentication; the service
info request explicitly uses no authentication. Do not export populated secrets.

For wallet queries, replace the router placeholder with a real user ID, for
example `/helpdesk/wallets/12345`, not `/helpdesk/wallets/:userId`.

Run repository and HTTP integration tests against a disposable local MongoDB:

```shell
MONGO_INTEGRATION_URI=mongodb://localhost:27017 make test
```

The tests create uniquely named databases and drop only those databases on
completion. Without `MONGO_INTEGRATION_URI`, MongoDB integration tests are skipped;
the unit and validator regression tests still run. Authentication tests cover
both keys, rejected headers, protected prefixes, startup configuration errors,
public probes and authentication before schema validation. HTTP integration
tests also verify that an unauthenticated reset does not modify the wallet.
The DLQ tests do not validate
the target Cosmos DB deployment or APIM configuration.

#### Performance testing

Performance tests are not yet automated.

## APIs

### Contract ID deactivation DLQ search

```shell
curl -i -H "x-api-key: ${SECURITY_API_KEYS_PRIMARY}" \
  'http://localhost:8080/helpdesk/wallets/contractId/deactivations/dlq?pageSize=10'
curl -i --get 'http://localhost:8080/helpdesk/wallets/contractId/deactivations/dlq' \
  -H "x-api-key: ${SECURITY_API_KEYS_PRIMARY}" \
  --data-urlencode 'pageSize=10' \
  --data-urlencode 'continuationCursor=<cursor returned by the previous page>'
```

`pageSize` is required and must be between 1 and 10. Results are ordered by DLQ
`timestamp` string (lexicographically) and `_id` descending, so the most recent
entries appear first. Reuse the opaque
`continuationCursor` for the
next page; an empty cursor in the response marks the final page. The cursor is a
versioned URL-safe base64 JSON cursor, not a credential or an authorization
boundary. Invalid cursors return 400. An empty page returns 404, including a
continuation page whose remaining entries have expired.
The cursor preserves the exact stored timestamp, including nanoseconds. Older
BSON Date-based cursors and v2 cursors from the previous ascending order are not
supported; restart pagination without a cursor.

The repository fetches at most `pageSize + 1` DLQ entries to detect another page
and queries error events for the returned wallet IDs in one bulk query (no
per-wallet queries or `$lookup`). Pagination is not a database snapshot: TTL
expiry, inserts or timestamp updates can change subsequent pages.

DLQ `_id` values and event `walletId` values are matched as strings; wallet IDs
do not need to be UUIDs. Event `_id` values must be canonical lowercase UUID
strings for their details to be included. The DLQ ID is the wallet ID, not an
event ID. The DLQ `timestamp` must be present as a string; non-RFC3339 strings
are returned and used literally for sorting and pagination, so their position
is deterministic but not necessarily chronological. For chronological DLQ
ordering, the producer should consistently use RFC3339 UTC (`Z`) and fixed
fractional-second precision. The compound `{ timestamp: 1, _id: 1 }` index can
serve the reverse scan. Event timestamps must be RFC3339 strings and are
parsed as instants for contract selection and error ordering, preserving
nanoseconds.
Events must
use `_class = it.pagopa.wallet.audit.WalletContractIdDeactivationErrorEvent`
and carry `contractId`, `errorType` and a 1-based `attemptNumber`.
`httpErrorCode` and `npgErrorCode` are optional and omitted from responses when
absent. The API exposes the NPG code as `npgErrorCode`, not `httpErrorDescription`.

Each result contains `id` and, when usable events exist, the latest event's
`contractId` and all recorded errors for that contract, ordered by attempt number,
timestamp and event ID. Latest contract selection uses timestamp, then attempt
number, then event ID as a stable tie-breaker. When there are no events, the
wallet is returned with `contractId` and `deactivationErrors` omitted. If event
data is malformed, the wallet is still returned with both detail fields omitted
so one bad entry does not block page traversal; the malformed data is logged.
Database failures return a logged 500 ProblemJson response without exposing
internal database errors to the client.

Before deployment, provision the collections and verify query support and indexes
on the target Cosmos DB MongoDB API. The DLQ needs a compound index supporting
`{ timestamp: 1, _id: 1 }`; individual indexes on those fields do not replace the
compound index required by Cosmos DB for a multi-field sort. The event collection
has `{ walletId: 1, timestamp: 1 }` (`walletId_1_timestamp_1`) and the default
`_id_` index. The bulk event lookup can use the `walletId` prefix; `_class` is
an additional filter, not part of that index. Verify query performance on the
target Cosmos DB instance. The service does not create or
modify indexes at startup. Agree on a uniform timestamp string format with the
producer before it starts writing records. On MongoDB, a TTL index cannot expire
documents based on a string timestamp; verify the target Cosmos DB TTL mechanism
separately rather than assuming date-based TTL behavior.

### Reset wallet deactivation retry counter

```shell
curl -i -X PATCH \
  -H "x-api-key: ${SECURITY_API_KEYS_PRIMARY}" \
  'http://localhost:8080/helpdesk/wallets/<walletId>/deactivationRetryCount'
```

The endpoint takes no request body. `walletId` must be a canonical lowercase UUID.
It atomically sets `deactivationRetryCount` to `0` in the `payment-wallets`
collection, without changing other fields. Any existing wallet can be updated,
regardless of status or DLQ membership. A missing counter is initialized to zero.
It does not modify the DLQ or directly trigger a deactivation attempt.

Success returns an empty HTTP 204 response, including when the counter was already
zero. An invalid wallet ID returns 400, a missing wallet returns 404 (no upsert),
and database errors return a logged 500 without exposing internal details.
Errors use `ProblemJson`. As with other wallet endpoints, the API key middleware
authenticates requests, with APIM enforcing additional access policies.

Successful resets are logged with the wallet ID, including requests where the
counter was already zero. This application log is not a persistent audit event.

The HTTP integration tests exercise the reset against disposable MongoDB databases,
including repeated requests, missing counters, missing wallets and preservation
of other wallet fields, other wallets and DLQ entries. Run them with the
`MONGO_INTEGRATION_URI` command above.

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
