FROM golang:1.22.6 AS builder
ARG CGO_ENABLED=0
ARG EXECUTABLE_NAME="payment-wallet-helpdesk-service"
WORKDIR /app

COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o ${EXECUTABLE_NAME}

FROM scratch
COPY --from=builder /app/${EXECUTABLE_NAME} /${EXECUTABLE_NAME}
ENTRYPOINT ["/payment-wallet-helpdesk-service"]