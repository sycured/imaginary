#
# SPDX-License-Identifier: AGPL-3.0-only
#
# Copyright (c) 2025 sycured
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, version 3.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.
#

FROM golang:1.24.3-alpine3.21@sha256:ef18ee7117463ac1055f5a370ed18b8750f01589f13ea0b48642f5792b234044
ARG GOLANGCILINT_VERSION=2.1.5
ARG VEGETA_VERSION=12.12.0

ENV CGO_ENABLED=1
ENV GO111MODULE=on

RUN addgroup -g 1000 nonroot \
    && adduser -u 1000 -G nonroot -D nonroot \
    && apk add --no-cache \
        bash=5.2.37-r0 \
        build-base=0.5-r3 \
        ca-certificates=20241121-r1 \
        curl=8.12.1-r1 \
        git=2.47.2-r0 \
        pkgconf=2.3.0-r0 \
        vips-dev=8.15.3-r5 \
    && curl --proto "=https" --tlsv1.2 -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GOPATH}/bin" v${GOLANGCILINT_VERSION} \
    && cpuarch=$(uname -m) \
    && [[ $cpuarch == x86_64 ]] && vegeta_arch=amd64 || vegeta_arch=arm \
    && curl --proto "=https" --tlsv1.2 -fsSLO https://github.com/tsenart/vegeta/releases/download/v${VEGETA_VERSION}/vegeta_${VEGETA_VERSION}_linux_${vegeta_arch}.tar.gz \
    && tar xf vegeta_${VEGETA_VERSION}_linux_${vegeta_arch}.tar.gz \
    && install -oroot -groot vegeta /usr/local/bin/vegeta \
    && rm vegeta_${VEGETA_VERSION}_linux_${vegeta_arch}.tar.gz

WORKDIR /workspace
