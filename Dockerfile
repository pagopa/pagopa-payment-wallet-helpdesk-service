FROM golang:1.22.6@sha256:a6322013fde74c44e925791b6f3fc4529c3bcd5982e09968ea14b6aa08309e05 AS builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY go.mod ./
RUN go mod download
COPY . .
RUN make build

FROM alpine:3.20.3@sha256:1e42bbe2508154c9126d48c2b8a75420c3544343bf86fd041fb7527e017a4b4a
ARG EXECUTABLE_NAME="pagopa-payment-wallet-helpdesk-service"
ARG EXECUTABLE_PATH="/tmp/bin/${EXECUTABLE_NAME}"
COPY --from=builder /app/api-spec  /api-spec
COPY --from=builder ${EXECUTABLE_PATH} ./${EXECUTABLE_NAME}
ENTRYPOINT ["./pagopa-payment-wallet-helpdesk-service"]
