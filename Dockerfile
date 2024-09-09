FROM golang:1.22.6 AS builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY go.mod ./
RUN go mod download
COPY . .
RUN make build

FROM alpine:3.20.3
ARG EXECUTABLE_NAME="pagopa-payment-wallet-helpdesk-service"
ARG EXECUTABLE_PATH="/tmp/bin/${EXECUTABLE_NAME}"
COPY --from=builder ${EXECUTABLE_PATH} ./${EXECUTABLE_NAME}
ENTRYPOINT ["./pagopa-payment-wallet-helpdesk-service"]
