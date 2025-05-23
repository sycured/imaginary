FROM cgr.dev/chainguard/go:latest-dev@sha256:6b6416597d91d4e1b5b8dc05cab1e7ef7e569c3f2a59d176f459ca11e1f5efef AS builder

ENV GOPATH=/go

ARG IMAGINARY_VERSION=dev

# Installs libvips + required libraries
RUN apk upgrade --no-cache --no-interactive \
    && apk add --no-cache --no-interactive ca-certificates libvips-dev

WORKDIR ${GOPATH}/src/github.com/sycured/imaginary


# Cache go modules
ENV GO111MODULE=on

COPY go.mod .
COPY go.sum .

RUN go mod download

# Copy imaginary sources
COPY . .

# Compile imaginary
RUN go build -a \
    -o ${GOPATH}/bin/imaginary \
    -ldflags="-s -w -h -X main.Version=${IMAGINARY_VERSION}" \
    -trimpath \
    github.com/sycured/imaginary


FROM cgr.dev/chainguard/wolfi-base:latest@sha256:0c35d31660ee8ff26c0893f7f1fe5752aea11f036536368791d2854e67112f85

ARG IMAGINARY_VERSION

LABEL maintainer="60801403+sycured@users.noreply.github.com" \
      org.label-schema.description="Fast, simple, scalable HTTP microservice for high-level image processing with first-class Docker support" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/sycured/imaginary" \
      org.label-schema.vcs-url="https://github.com/sycured/imaginary" \
      org.label-schema.version="${IMAGINARY_VERSION}"

COPY --from=builder --chown=root:root --chmod=755 /go/bin/imaginary /usr/local/bin/imaginary

# Install runtime dependencies
RUN apk upgrade --no-cache --no-interactive \
    && apk add --no-cache --no-interactive ca-certificates jemalloc libvips \
    && ln -s /usr/lib/libjemalloc.so.2 /usr/local/lib/libjemalloc.so \
    && rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

ENV LD_PRELOAD=/usr/local/lib/libjemalloc.so

# Server port to listen
ENV PORT=9000

# Drop privileges for non-UID mapped environments
USER nobody

# Run the entrypoint command by default when the container starts.
ENTRYPOINT ["/usr/local/bin/imaginary"]

# Expose the server TCP port
EXPOSE ${PORT}
