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
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	d "runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/gopkg/util/gctuner"
	"github.com/h2non/bimg"
)

var (
	aAddr               = flag.String("a", "", "Bind address")
	aPort               = flag.Int("p", 8088, "Port to listen")
	aVers               = flag.Bool("v", false, "Show version")
	aVersl              = flag.Bool("version", false, "Show version")
	aHelp               = flag.Bool("h", false, "Show help")
	aHelpl              = flag.Bool("help", false, "Show help")
	aPathPrefix         = flag.String("path-prefix", "/", "Url path prefix to listen to")
	aCors               = flag.Bool("cors", false, "Enable CORS support")
	aGzip               = flag.Bool("gzip", false, "Enable gzip compression (deprecated)")
	aAuthForwarding     = flag.Bool("enable-auth-forwarding", false, "Forwards X-Forward-Authorization or Authorization header to the image source server. -enable-url-source flag must be defined. Tip: secure your server from public access to prevent attack vectors") //nolint:lll
	aEnableURLSource    = flag.Bool("enable-url-source", false, "Enable remote HTTP URL image source processing")
	aAllowInsecureSSL   = flag.Bool("insecure", false, "Allow connections to endpoints with insecure SSL certificates. -enable-url-source flag must be defined. Note: Should only be used in development.") //nolint:lll
	aEnablePlaceholder  = flag.Bool("enable-placeholder", false, "Enable image response placeholder to be used in case of error")                                                                           //nolint:lll
	aEnableURLSignature = flag.Bool("enable-url-signature", false, "Enable URL signature (URL-safe Base64-encoded HMAC digest)")                                                                            //nolint:lll
	aURLSignatureKey    = flag.String("url-signature-key", "", "The URL signature key (32 characters minimum)")
	aAllowedOrigins     = flag.String("allowed-origins", "", "Restrict remote image source processing to certain origins (separated by commas). Note: Origins are validated against host *AND* path.") //nolint:lll
	aMaxAllowedSize     = flag.Int("max-allowed-size", 0, "Restrict maximum size of http image source (in bytes)")                                                                                     //nolint:lll
	aMaxAllowedPixels   = flag.Float64("max-allowed-resolution", 18.0, "Restrict maximum resolution of the image (in megapixels)")                                                                     //nolint:lll
	aKey                = flag.String("key", "", "Define API key for authorization")
	aMount              = flag.String("mount", "", "Mount server local directory")
	aCertFile           = flag.String("certfile", "", "TLS certificate file path")
	aKeyFile            = flag.String("keyfile", "", "TLS private key file path")
	aAuthorization      = flag.String("authorization", "", "Defines a constant Authorization header value passed to all the image source servers. -enable-url-source flag must be defined. This overwrites authorization headers forwarding behavior via X-Forward-Authorization")                                                                                                                                        //nolint:lll
	aForwardHeaders     = flag.String("forward-headers", "", "Forwards custom headers to the image source server. -enable-url-source flag must be defined.")                                                                                                                                                                                                                                                              //nolint:lll
	aSrcResponseHeaders = flag.String("source-response-headers", "", "Returns selected headers from the source image server response. Has precedence over -http-cache-ttl when cache-control is specified and the source response has a cache-control header, otherwise falls back to -http-cache-ttl value if provided. Missing and/or unlisted response headers are ignored. -enable-url-source flag must be defined.") //nolint:lll
	aPlaceholder        = flag.String("placeholder", "", "Image path to image custom placeholder to be used in case of error. Recommended minimum image size is: 1200x1200")                                                                                                                                                                                                                                              //nolint:lll
	aPlaceholderStatus  = flag.Int("placeholder-status", 0, "HTTP status returned when use -placeholder flag")
	aDisableEndpoints   = flag.String("disable-endpoints", "", "Comma separated endpoints to disable. E.g: form,crop,rotate,health") //nolint:lll
	aHTTPCacheTTL       = flag.Int("http-cache-ttl", -1, "The TTL in seconds")
	aReadTimeout        = flag.Int("http-read-timeout", 60, "HTTP read timeout in seconds")
	aWriteTimeout       = flag.Int("http-write-timeout", 60, "HTTP write timeout in seconds")
	aConcurrency        = flag.Int("concurrency", 0, "Throttle concurrency limit per second")
	aBurst              = flag.Int("burst", 100, "Throttle burst max cache size")
	aMRelease           = flag.Int("mrelease", 30, "OS memory release interval in seconds")
	aLogLevel           = flag.String("log-level", "info", "Define log level for http-server. E.g: info,warning,error")
	aReturnSize         = flag.Bool("return-size", false, "Return the image size in the HTTP headers")
)

//nolint:lll
const usage = `imaginary %s

Usage:
  imaginary -p 80
  imaginary -cors
  imaginary -concurrency 10
  imaginary -path-prefix /api/v1
  imaginary -enable-url-source
  imaginary -disable-endpoints form,health,crop,rotate
  imaginary -enable-url-source -allowed-origins http://localhost,http://server.com
  imaginary -enable-url-source -enable-auth-forwarding
  imaginary -enable-url-source -authorization "Basic AwDJdL2DbwrD=="
  imaginary -enable-placeholder
  imaginary -enable-url-source -placeholder ./placeholder.jpg
  imaginary -enable-url-signature -url-signature-key 4f46feebafc4b5e988f131c4ff8b5997
  imaginary -enable-url-source -forward-headers X-Custom,X-Token
  imaginary -h | -help
  imaginary -v | -version

Options:

  -a <addr>                            Bind address [default: *]
  -p <port>                            Bind port [default: 8088]
  -h, -help                            Show help
  -v, -version                         Show version
  -path-prefix <value>                 Url path prefix to listen to [default: "/"]
  -cors                                Enable CORS support [default: false]
  -gzip                                Enable gzip compression (deprecated) [default: false]
  -disable-endpoints                   Comma separated endpoints to disable. E.g: form,crop,rotate,health [default: ""]
  -key <key>                           Define API key for authorization
  -mount <path>                        Mount server local directory
  -http-cache-ttl <num>                The TTL in seconds. Adds caching headers to locally served files.
  -http-read-timeout <num>             HTTP read timeout in seconds [default: 30]
  -http-write-timeout <num>            HTTP write timeout in seconds [default: 30]
  -enable-url-source                   Enable remote HTTP URL image source processing
  -insecure                            Allow connections to endpoints with insecure SSL certificates.
                                       -enable-url-source flag must be defined.
                                       Note: Should only be used in development.
  -enable-placeholder                  Enable image response placeholder to be used in case of error [default: false]
  -enable-auth-forwarding              Forwards X-Forward-Authorization or Authorization header to the image source server. -enable-url-source flag must be defined. Tip: secure your server from public access to prevent attack vectors
  -forward-headers                     Forwards custom headers to the image source server. -enable-url-source flag must be defined.
  -source-response-headers             Returns selected headers from the source image server response. Has precedence over -http-cache-ttl when cache-control is specified and the source response has a cache-control header, otherwise falls back to -http-cache-ttl value if provided. Missing and/or unlisted response headers are ignored. -enable-url-source flag must be defined.
  -enable-url-signature                Enable URL signature (URL-safe Base64-encoded HMAC digest) [default: false]
  -url-signature-key                   The URL signature key (32 characters minimum)
  -allowed-origins <urls>              Restrict remote image source processing to certain origins (separated by commas)
  -max-allowed-size <bytes>            Restrict maximum size of http image source (in bytes)
  -max-allowed-resolution <megapixels> Restrict maximum resolution of the image [default: 18.0]
  -certfile <path>                     TLS certificate file path
  -keyfile <path>                      TLS private key file path
  -authorization <value>               Defines a constant Authorization header value passed to all the image source servers. -enable-url-source flag must be defined. This overwrites authorization headers forwarding behavior via X-Forward-Authorization
  -placeholder <path>                  Image path to image custom placeholder to be used in case of error. Recommended minimum image size is: 1200x1200
  -placeholder-status <code>           HTTP status returned when use -placeholder flag
  -concurrency <num>                   Throttle concurrency limit per second [default: disabled]
  -burst <num>                         Throttle burst max cache size [default: 100]
  -mrelease <num>                      OS memory release interval in seconds [default: 30]
  -cpus <num>                          Number of used cpu cores.
                                       (default for current machine is %d cores)
  -log-level                           Set log level for http-server. E.g: info,warning,error [default: info].
                                       Or can use the environment variable GOLANG_LOG=info.
  -return-size                         Return the image size with X-Width and X-Height HTTP header. [default: disabled].
`

type URLSignature struct {
	Key string
}

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, usage, Version, runtime.NumCPU())
	}
	flag.Parse()

	if *aHelp || *aHelpl {
		showUsage()
	}
	if *aVers || *aVersl {
		showVersion()
	}

	memoryLimit := getMemoryLimit()
	if memoryLimit == 0 {
		log.Panicf("Failed to determine host memory limit")
		return
	}

	var gcThresholdCoeff = 0.7
	if val, ok := os.LookupEnv("GCTHRESHOLDCOEFF"); ok && val != "" {
		if parsedVal, err := strconv.ParseFloat(val, 64); err == nil {
			gcThresholdCoeff = parsedVal
		}
	}

	gcThreshold := float64(memoryLimit) * gcThresholdCoeff
	gctuner.Tuning(uint64(gcThreshold))

	port := getPort(*aPort)
	urlSignature := getURLSignature(*aURLSignatureKey)

	opts := createServerOptions(port, urlSignature)

	handleDeprecationWarnings()
	configureMemoryRelease()
	validateMountDirectory()
	validateCacheTTL(opts)
	managePlaceholderImage(&opts)
	validateURLSignatureKey(urlSignature, opts)

	debug("imaginary server listening on port :%d/%s", opts.Port, strings.TrimPrefix(opts.PathPrefix, "/"))

	// Load image source providers and start the server
	LoadSources(opts)
	Server(opts)
}

// createServerOptions initializes the ServerOptions
func createServerOptions(port int, urlSignature URLSignature) ServerOptions {
	return ServerOptions{
		Port:               port,
		Address:            *aAddr,
		CORS:               *aCors,
		AuthForwarding:     *aAuthForwarding,
		EnableURLSource:    *aEnableURLSource,
		AllowInsecureSSL:   *aAllowInsecureSSL,
		EnablePlaceholder:  *aEnablePlaceholder,
		EnableURLSignature: *aEnableURLSignature,
		URLSignatureKey:    urlSignature.Key,
		PathPrefix:         *aPathPrefix,
		APIKey:             *aKey,
		Concurrency:        *aConcurrency,
		Burst:              *aBurst,
		Mount:              *aMount,
		CertFile:           *aCertFile,
		KeyFile:            *aKeyFile,
		Placeholder:        *aPlaceholder,
		PlaceholderStatus:  *aPlaceholderStatus,
		HTTPCacheTTL:       *aHTTPCacheTTL,
		HTTPReadTimeout:    *aReadTimeout,
		HTTPWriteTimeout:   *aWriteTimeout,
		Authorization:      *aAuthorization,
		ForwardHeaders:     parseHeadersList(*aForwardHeaders),
		SrcResponseHeaders: parseHeadersList(*aSrcResponseHeaders),
		AllowedOrigins:     parseOrigins(*aAllowedOrigins),
		MaxAllowedSize:     *aMaxAllowedSize,
		MaxAllowedPixels:   *aMaxAllowedPixels,
		LogLevel:           getLogLevel(*aLogLevel),
		ReturnSize:         *aReturnSize,
		Endpoints:          parseEndpoints(*aDisableEndpoints),
	}
}

// handleDeprecationWarnings handles deprecated flags
func handleDeprecationWarnings() {
	if *aGzip {
		fmt.Println("warning: -gzip flag is deprecated and will not have effect")
	}
}

// configureMemoryRelease sets up a memory release goroutine if necessary
func configureMemoryRelease() {
	if *aMRelease > 0 {
		memoryRelease(*aMRelease)
	}
}

// validateMountDirectory checks if the mount directory exists
func validateMountDirectory() {
	if *aMount != "" {
		checkMountDirectory(*aMount)
	}
}

// validateCacheTTL checks the HTTP cache parameter
func validateCacheTTL(opts ServerOptions) {
	if opts.HTTPCacheTTL != -1 {
		checkHTTPCacheTTL(opts.HTTPCacheTTL)
	}
}

// managePlaceholderImage configures the placeholder image
func managePlaceholderImage(opts *ServerOptions) {
	if *aPlaceholder != "" {
		buf, err := os.ReadFile(*aPlaceholder)
		if err != nil {
			exitWithError("cannot start the server: %s", err)
		}

		imageType := bimg.DetermineImageType(buf)
		if !bimg.IsImageTypeSupportedByVips(imageType).Load {
			exitWithError("Placeholder image type is not supported. Only JPEG, PNG or WEBP are supported")
		}

		opts.PlaceholderImage = buf
	} else if opts.EnablePlaceholder {
		// Expose default placeholder
		opts.PlaceholderImage = placeholder
	}
}

// validateURLSignatureKey checks the URL signature key if required
func validateURLSignatureKey(urlSignature URLSignature, opts ServerOptions) {
	if opts.EnableURLSignature {
		if urlSignature.Key == "" {
			exitWithError("URL signature key is required")
		}

		if len(urlSignature.Key) < 32 {
			exitWithError("URL signature key must be a minimum of 32 characters")
		}
	}
}

func getPort(port int) int {
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		newPort, _ := strconv.Atoi(portEnv)
		if newPort > 0 {
			port = newPort
		}
	}
	return port
}

func getURLSignature(key string) URLSignature {
	if keyEnv := os.Getenv("URL_SIGNATURE_KEY"); keyEnv != "" {
		key = keyEnv
	}

	return URLSignature{key}
}

func getLogLevel(logLevel string) string {
	if logLevelEnv := os.Getenv("GOLANG_LOG"); logLevelEnv != "" {
		logLevel = logLevelEnv
	}
	return logLevel
}

func showUsage() {
	flag.Usage()
	os.Exit(1)
}

func showVersion() {
	fmt.Println(Version)
	os.Exit(1)
}

func checkMountDirectory(path string) {
	src, err := os.Stat(path)
	if err != nil {
		exitWithError("error while mounting directory: %s", err)
	}
	if !src.IsDir() {
		exitWithError("mount path is not a directory: %s", path)
	}
	if path == "/" {
		exitWithError("cannot mount root directory for security reasons")
	}
}

func checkHTTPCacheTTL(ttl int) {
	if ttl < 0 || ttl > 31556926 {
		exitWithError("The -http-cache-ttl flag only accepts a value from 0 to 31556926")
	}

	if ttl == 0 {
		debug("Adding HTTP cache control headers set to prevent caching.")
	}
}

func parseHeadersList(headerString string) []string {
	var headers []string
	if headerString == "" {
		return headers
	}

	for _, header := range strings.Split(headerString, ",") {
		if norm := strings.TrimSpace(header); norm != "" {
			headers = append(headers, norm)
		}
	}
	return headers
}

func parseOrigins(origins string) []*url.URL {
	urls := make([]*url.URL, 0, 10)
	if origins == "" {
		return urls
	}
	for _, origin := range strings.Split(origins, ",") {
		u, err := url.Parse(origin)
		if err != nil {
			continue
		}

		if u.Path != "" {
			var lastChar = u.Path[len(u.Path)-1:]
			if lastChar == "*" {
				u.Path = strings.TrimSuffix(u.Path, "*")
			} else if lastChar != "/" {
				u.Path += "/"
			}
		}

		urls = append(urls, u)
	}
	return urls
}

func parseEndpoints(input string) Endpoints {
	var endpoints Endpoints
	for _, endpoint := range strings.Split(input, ",") {
		endpoint = strings.ToLower(strings.TrimSpace(endpoint))
		if endpoint != "" {
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func memoryRelease(interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	go func() {
		for range ticker.C {
			debug("FreeOSMemory()")
			d.FreeOSMemory()
		}
	}()
}

func exitWithError(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args)
	os.Exit(1)
}

func debug(msg string, values ...interface{}) {
	debug := os.Getenv("DEBUG")
	if debug == "imaginary" || debug == "*" {
		log.Printf(msg, values...)
	}
}
