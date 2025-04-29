/*
 * SPDX-License-Identifier: AGPL-3.0-only
 *
 * Copyright (c) 2025 sycured
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, version 3.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/swaggo/http-swagger"
	_ "github.com/sycured/imaginary/docs"
)

type ServerOptions struct {
	Port               int
	QUICPort           int
	QUICPublicPort     int
	Burst              int
	Concurrency        int
	HTTPCacheTTL       int
	HTTPReadTimeout    int
	HTTPWriteTimeout   int
	MaxAllowedSize     int
	MaxAllowedPixels   float64
	CORS               bool
	Gzip               bool // deprecated
	AuthForwarding     bool
	EnableURLSource    bool
	AllowInsecureSSL   bool
	EnablePlaceholder  bool
	EnableURLSignature bool
	URLSignatureKey    string
	Address            string
	PathPrefix         string
	APIKey             string
	Mount              string
	CertFile           string
	KeyFile            string
	Authorization      string
	Placeholder        string
	PlaceholderStatus  int
	ForwardHeaders     []string
	SrcResponseHeaders []string
	PlaceholderImage   []byte
	Endpoints          Endpoints
	AllowedOrigins     []*url.URL
	LogLevel           string
	ReturnSize         bool
}

// Endpoints represents a list of endpoint names to disable.
type Endpoints []string

// IsValid validates if a given HTTP request endpoint is valid or not.
func (e Endpoints) IsValid(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")
	endpoint := parts[len(parts)-1]
	for _, name := range e {
		if endpoint == name {
			return false
		}
	}
	return true
}

// setupTLSConfig creates and returns the TLS configuration if certificates are provided
func setupTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load X509 key pair: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// createHTTPServer creates an HTTP/HTTPS server with the given handler and options
func createHTTPServer(addr string, handler http.Handler, o ServerOptions, tlsConfig *tls.Config) *http.Server {
	srv := &http.Server{
		Addr:           addr,
		Handler:        altSvcMiddleware(handler, o.QUICPort),
		MaxHeaderBytes: 1 << 20,
		ReadTimeout:    time.Duration(o.HTTPReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(o.HTTPWriteTimeout) * time.Second,
		TLSConfig:      tlsConfig,
	}
	if o.QUICPublicPort != 0 {
		srv.Handler = altSvcMiddleware(handler, o.QUICPublicPort)
	} else {
		srv.Handler = altSvcMiddleware(handler, o.QUICPort)
	}

	return srv
}

// createHTTP3Server creates an HTTP/3 server if TLS is configured
func createHTTP3Server(quicAddr string, handler http.Handler, tlsConfig *tls.Config, port int, qPP int) *http3.Server {
	if tlsConfig == nil {
		return nil
	}

	h3Server := &http3.Server{
		Addr:      quicAddr,
		Handler:   handler,
		TLSConfig: http3.ConfigureTLSConfig(tlsConfig),
		QUICConfig: &quic.Config{
			MaxIdleTimeout: 30 * time.Second,
			Allow0RTT:      false,
		},
	}

	if qPP != 0 {
		h3Server.Port = qPP
	} else {
		h3Server.Port = port
	}

	return h3Server
}

// startHTTPServer starts the HTTP/HTTPS server in a goroutine
func startHTTPServer(server *http.Server, certFile, keyFile string) {
	go func() {
		var err error
		if certFile != "" && keyFile != "" {
			log.Printf("Starting HTTPS server on %s", server.Addr)
			err = server.ListenAndServeTLS(certFile, keyFile)
		} else {
			log.Printf("Starting HTTP server on %s", server.Addr)
			err = server.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP(S) server error: %s\n", err)
		}
	}()
}

// startHTTP3Server starts the HTTP/3 server in a goroutine if it exists
func startHTTP3Server(server *http3.Server) {
	if server == nil {
		return
	}

	go func() {
		log.Printf("Starting HTTP/3 server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("HTTP/3 server error: %s\n", err)
		}
	}()
}

// Server sets up and starts the HTTP and HTTP/3 servers
func Server(o ServerOptions) {
	addr := o.Address + ":" + strconv.Itoa(o.Port)
	quicAddr := o.Address + ":" + strconv.Itoa(o.QUICPort)

	// Create the base handler
	baseHandler := NewLog(NewServerMux(o), os.Stdout, o.LogLevel)
	handler := baseHandler

	// Setup TLS if certificates are provided
	tlsConfig, err := setupTLSConfig(o.CertFile, o.KeyFile)
	if err != nil {
		log.Panic(err)
	}

	// Create servers
	http3Server := createHTTP3Server(quicAddr, baseHandler, tlsConfig, o.QUICPort, o.QUICPublicPort)
	httpServer := createHTTPServer(addr, handler, o, tlsConfig)

	// Start servers
	startHTTPServer(httpServer, o.CertFile, o.KeyFile)
	startHTTP3Server(http3Server)

	// Setup graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done
	log.Print("Graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown failed: %+v", err)
	}

	log.Print("Server shutdown completed")
}

// @Summary Prometheus metrics
// @Description Returns Prometheus metrics for monitoring
// @Produce text/plain
// @Success 200 {string} string "Prometheus metrics"
// @Router /metrics [get]
func metricsHandler() http.Handler {
	return promhttp.Handler()
}

func altSvcMiddleware(h http.Handler, quicPort int) http.Handler {
	// Format with full hostname and port
	altSvcValue := fmt.Sprintf(`h3=":%d"; ma=2592000`, quicPort)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Alt-Svc", altSvcValue)
		h.ServeHTTP(w, r)
	})
}

func join(o ServerOptions, route string) string {
	return path.Join(o.PathPrefix, route)
}

// NewServerMux creates a new HTTP server route multiplexer.
func NewServerMux(o ServerOptions) http.Handler {
	mux := http.NewServeMux()

	mux.Handle(join(o, "/"), Middleware(indexController(o), o))
	mux.Handle(join(o, "/form"), Middleware(formController(o), o))
	mux.Handle(join(o, "/health"), Middleware(healthController, o))
	mux.Handle(join(o, "/metrics"), metricsHandler())
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	image := ImageMiddleware(o)
	mux.Handle(join(o, "/autorotate"), image(AutoRotate))
	mux.Handle(join(o, "/blur"), image(GaussianBlur))
	mux.Handle(join(o, "/convert"), image(Convert))
	mux.Handle(join(o, "/crop"), image(Crop))
	mux.Handle(join(o, "/enlarge"), image(Enlarge))
	mux.Handle(join(o, "/extract"), image(Extract))
	mux.Handle(join(o, "/fit"), image(Fit))
	mux.Handle(join(o, "/flip"), image(Flip))
	mux.Handle(join(o, "/flop"), image(Flop))
	mux.Handle(join(o, "/info"), image(Info))
	mux.Handle(join(o, "/pipeline"), image(Pipeline))
	mux.Handle(join(o, "/resize"), image(Resize))
	mux.Handle(join(o, "/rotate"), image(Rotate))
	mux.Handle(join(o, "/smartcrop"), image(SmartCrop))
	mux.Handle(join(o, "/thumbnail"), image(Thumbnail))
	mux.Handle(join(o, "/watermark"), image(Watermark))
	mux.Handle(join(o, "/watermarkimage"), image(WatermarkImage))
	mux.Handle(join(o, "/zoom"), image(Zoom))

	return mux
}
