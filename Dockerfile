ARG GOLANG_VERSION=1.24
FROM golang:${GOLANG_VERSION}-bookworm AS builder

ARG IMAGINARY_VERSION=dev
ARG GOLANGCILINT_VERSION=1.64.7

# Installs libvips + required libraries
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get dist-upgrade -y && \
  apt-get install --no-install-recommends -y \
    automake \
    build-essential \
    ca-certificates \
    curl \
    gobject-introspection \
    gtk-doc-tools \
    libcfitsio-dev \
    libexif-dev \
    libfftw3-dev \
    libgif-dev \
    libglib2.0-dev \
    libgsf-1-dev \
    libheif-dev \
    libimagequant-dev \
    libjpeg62-turbo-dev \
    libmagickwand-dev \
    libmatio-dev \
    libopenslide-dev \
    liborc-0.4-dev \
    libpango1.0-dev \
    libpng-dev \
    libpoppler-glib-dev \
    librsvg2-dev \
    libtiff-dev \
    libvips-dev \
    libwebp-dev \
    libxml2-dev \
    swig

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

FROM debian:bookworm-slim@sha256:b1211f6d19afd012477bd34fdcabb6b663d680e0f4b0537da6e6b0fd057a3ec3

ARG IMAGINARY_VERSION

LABEL maintainer="60801403+sycured@users.noreply.github.com" \
      org.label-schema.description="Fast, simple, scalable HTTP microservice for high-level image processing with first-class Docker support" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/sycured/imaginary" \
      org.label-schema.vcs-url="https://github.com/sycured/imaginary" \
      org.label-schema.version="${IMAGINARY_VERSION}"

COPY --from=builder /usr/local/lib /usr/local/lib
COPY --from=builder /go/bin/imaginary /usr/local/bin/imaginary
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

# Install runtime dependencies
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get dist-upgrade -y && \
  apt-get install --no-install-recommends -y \
   libcfitsio10 \
   libexif12 \
   libfftw3-bin \
   libgif7 \
   libglib2.0-0 \
   libgsf-1-114 \
   libheif1 \
   libimagequant0 \
   libjemalloc2 \
   libjpeg62-turbo \
   libmagickwand-6.q16-6 \
   libmatio11 \
   libopenexr-3-1-30 \
   libopenslide0 \
   liborc-0.4-0 \
   libpango1.0-0 \
   libpng16-16 \
   libpoppler-glib8 \
   librsvg2-2 \
   libtiff6 \
   libvips42 \
   libwebp7 \
   libwebpdemux2 \
   libwebpmux3 \
   libxml2 \
   procps && \
  ln -s /usr/lib/$(uname -m)-linux-gnu/libjemalloc.so.2 /usr/local/lib/libjemalloc.so && \
  apt-get autoremove -y && \
  apt-get autoclean && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
ENV LD_PRELOAD=/usr/local/lib/libjemalloc.so

# Server port to listen
ENV PORT=9000

# Drop privileges for non-UID mapped environments
USER nobody

# Run the entrypoint command by default when the container starts.
ENTRYPOINT ["/usr/local/bin/imaginary"]

# Expose the server TCP port
EXPOSE ${PORT}
