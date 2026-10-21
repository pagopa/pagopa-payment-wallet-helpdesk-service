FROM golang:1.27.3@sha256:957861185dbf29d93668ace41e011d091c6e779953d47f0316ef2cc58e28fe2b AS builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
ARG EXECUTABLE_NAME="pagopa-payment-wallet-helpdesk-service"
ARG EXECUTABLE_PATH="/tmp/bin/${EXECUTABLE_NAME}"
COPY --from=builder /app/api-spec  /api-spec
COPY --from=builder ${EXECUTABLE_PATH} ./${EXECUTABLE_NAME}
FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

ARG EXECUTABLE_NAME="pagopa-payment-wallet-helpdesk-service"
ARG EXECUTABLE_PATH="/tmp/bin/${EXECUTABLE_NAME}"

RUN addgroup --system --gid 10001 user && adduser --ingroup user --system --uid 10001 user
COPY --from=builder /app/api-spec  /api-spec
COPY --from=builder ${EXECUTABLE_PATH} ./${EXECUTABLE_NAME}

USER 10001:10001
ENTRYPOINT ["./pagopa-payment-wallet-helpdesk-service"]
