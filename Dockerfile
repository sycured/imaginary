ARG GOLANG_VERSION=1.24
FROM golang:${GOLANG_VERSION}-bullseye AS builder

ARG IMAGINARY_VERSION=dev
ARG LIBVIPS_VERSION=8.16.0
ARG GOLANGCILINT_VERSION=1.64.6

# Installs libvips + required libraries
ADD https://github.com/libvips/libvips/releases/download/v${LIBVIPS_VERSION}/vips-${LIBVIPS_VERSION}.tar.xz /tmp/vips-${LIBVIPS_VERSION}.tar.xz
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
    automake build-essential ca-certificates curl fftw3-dev gobject-introspection gtk-doc-tools libcfitsio-dev \
    libexif-dev libgif-dev libglib2.0-dev libgsf-1-dev libheif-dev libimagequant-dev libjpeg62-turbo-dev \
    libmagickwand-dev libmatio-dev libopenslide-dev liborc-0.4-dev libpango1.0-dev libpng-dev libpoppler-glib-dev \
    librsvg2-dev libtiff5-dev libwebp-dev libxml2-dev swig && \
  cd /tmp && \
  tar vxf vips-${LIBVIPS_VERSION}.tar.xz && \
  cd /tmp/vips-${LIBVIPS_VERSION} && \
	CFLAGS="-g -O3" CXXFLAGS="-D_GLIBCXX_USE_CXX11_ABI=0 -g -O3" \
    ./configure \
    --disable-debug \
    --disable-dependency-tracking \
    --disable-introspection \
    --disable-static \
    --enable-gtk-doc-html=no \
    --enable-gtk-doc=no \
    --enable-pyvips8=no && \
  make && make install && ldconfig && \
  rm -rf vips-${LIBVIPS_VERSION}.tar.xz vips-${LIBVIPS_VERSION}

# Installing golangci-lint
WORKDIR /tmp
RUN curl --proto "=https" --tlsv1.2 -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GOPATH}/bin" v${GOLANGCILINT_VERSION}

WORKDIR ${GOPATH}/src/github.com/sycured/imaginary

# Cache go modules
ENV GO111MODULE=on

COPY go.mod .
COPY go.sum .

RUN go mod download

# Copy imaginary sources
COPY . .

# Run quality control
ARG TARGETPLATFORM
RUN if [ "$TARGETPLATFORM" = "linux/arm64" ] ; then \
  go test ./... -test.v -test.coverprofile=atomic . ; \
  else go test ./... -test.v -race -test.coverprofile=atomic . ; \
  fi; \
  golangci-lint run .

# Compile imaginary
RUN go build -a \
    -o ${GOPATH}/bin/imaginary \
    -ldflags="-s -w -h -X main.Version=${IMAGINARY_VERSION}" \
    github.com/h2non/imaginary

FROM debian:bullseye-slim

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
  apt-get install --no-install-recommends -y \
   fftw3 libcfitsio9 libexif12 libgif7 libglib2.0-0 libgsf-1-114 libheif1 libimagequant0 libjemalloc2 libjpeg62-turbo \
   libmagickwand-6.q16-6 libmatio11 libopenexr25 libopenslide0 liborc-0.4-0 libpango1.0-0 libpng16-16 libpoppler-glib8 \
   librsvg2-2 libtiff5 libwebp6 libwebpdemux2 libwebpmux3 libxml2 procps && \
  ln -s /usr/lib/$(uname -m)-linux-gnu/libjemalloc.so.2 /usr/local/lib/libjemalloc.so && \
  apt-get autoremove -y && \
  apt-get autoclean && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
ENV LD_PRELOAD=/usr/local/lib/libjemalloc.so

# Server port to listen
ENV PORT 9000

# Drop privileges for non-UID mapped environments
USER nobody

# Run the entrypoint command by default when the container starts.
ENTRYPOINT ["/usr/local/bin/imaginary"]

# Expose the server TCP port
EXPOSE ${PORT}
