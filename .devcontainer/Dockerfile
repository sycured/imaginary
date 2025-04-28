FROM golang:alpine
ARG GOLANGCILINT_VERSION=2.1.5
ARG VEGETA_VERSION=12.12.0

ENV CGO_ENABLED=1
ENV GO111MODULE=on

RUN addgroup -g 1000 nonroot \
    && adduser -u 1000 -G nonroot -D nonroot \
    && apk add --no-cache ca-certificates bash build-base curl git make vips-dev pkgconf \
    && curl --proto "=https" --tlsv1.2 -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GOPATH}/bin" v${GOLANGCILINT_VERSION} \
    && cpuarch=$(uname -m) \
    && [[ $cpuarch == x86_64 ]] && vegeta_arch=amd64 || vegeta_arch=arm \
    && curl --proto "=https" --tlsv1.2 -fsSLO https://github.com/tsenart/vegeta/releases/download/v${VEGETA_VERSION}/vegeta_${VEGETA_VERSION}_linux_${vegeta_arch}.tar.gz \
    && tar xf vegeta_${VEGETA_VERSION}_linux_${vegeta_arch}.tar.gz \
    && install -oroot -groot vegeta /usr/local/bin/vegeta \
    && rm vegeta_${VEGETA_VERSION}_linux_${vegeta_arch}.tar.gz

WORKDIR /workspace
