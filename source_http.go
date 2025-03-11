package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const ImageSourceTypeHTTP ImageSourceType = "http"
const URLQueryKey = "url"

type HTTPImageSource struct {
	Config *SourceConfig
}

func NewHTTPImageSource(config *SourceConfig) ImageSource {
	return &HTTPImageSource{config}
}

func (s *HTTPImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodGet && r.URL.Query().Get(URLQueryKey) != ""
}

func (s *HTTPImageSource) GetImage(req *http.Request) ([]byte, error) {
	u, err := parseURL(req)
	if err != nil {
		return nil, ErrInvalidImageURL
	}
	if shouldRestrictOrigin(u, s.Config.AllowedOrigins) {
		return nil, fmt.Errorf("not allowed remote URL origin: %s%s", u.Host, u.Path)
	}
	return s.fetchImage(u, req)
}

func (s *HTTPImageSource) fetchImage(url *url.URL, ireq *http.Request) ([]byte, error) {
	// Check remote image size by fetching HTTP Headers
	if s.Config.MaxAllowedSize > 0 {
		req := newHTTPRequest(s, ireq, http.MethodHead, url)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error fetching remote http image headers: %v", err)
		}
		_ = res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode > 206 {
			return nil, NewError(fmt.Sprintf(
				"error fetching remote http image headers: (status=%d) (url=%s)", res.StatusCode, req.URL.String()), res.StatusCode)
		}

		contentLength, _ := strconv.Atoi(res.Header.Get("Content-Length"))
		if contentLength > s.Config.MaxAllowedSize {
			return nil, fmt.Errorf("Content-Length %d exceeds maximum allowed %d bytes", contentLength, s.Config.MaxAllowedSize)
		}
	}

	// Perform the request using the default client
	req := newHTTPRequest(s, ireq, http.MethodGet, url)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching remote http image: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)
	if res.StatusCode != 200 {
		return nil, NewError(
			fmt.Sprintf("error fetching remote http image: (status=%d) (url=%s)", res.StatusCode, req.URL.String()), res.StatusCode) //nolint:lll
	}

	// Read the body
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to create image from response body: %s (url=%s)", req.URL.String(), err)
	}
	return buf, nil
}

func (s *HTTPImageSource) setAuthorizationHeader(req *http.Request, ireq *http.Request) {
	auth := s.Config.Authorization
	if auth == "" {
		auth = ireq.Header.Get("X-Forward-Authorization")
	}
	if auth == "" {
		auth = ireq.Header.Get("Authorization")
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
}

func (s *HTTPImageSource) setForwardHeaders(req *http.Request, ireq *http.Request) {
	headers := s.Config.ForwardHeaders
	for _, header := range headers {
		if _, ok := ireq.Header[header]; ok {
			req.Header.Set(header, ireq.Header.Get(header))
		}
	}
}

func parseURL(request *http.Request) (*url.URL, error) {
	return url.Parse(request.URL.Query().Get(URLQueryKey))
}

func newHTTPRequest(s *HTTPImageSource, ireq *http.Request, method string, url *url.URL) *http.Request {
	req, _ := http.NewRequest(method, url.String(), nil)
	req.Header.Set("User-Agent", "imaginary/"+Version)
	req.URL = url

	if len(s.Config.ForwardHeaders) != 0 {
		s.setForwardHeaders(req, ireq)
	}

	// Forward auth header to the target server, if necessary
	if s.Config.AuthForwarding || s.Config.Authorization != "" {
		s.setAuthorizationHeader(req, ireq)
	}

	if s.Config.AllowInsecureSSL {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	return req
}

func shouldRestrictOrigin(url *url.URL, origins []*url.URL) bool {
	if len(origins) == 0 {
		return false
	}

	for _, origin := range origins {
		if isExactMatch(url, origin) || isSubdomainMatch(url, origin) {
			return false
		}
	}

	return true
}

func isExactMatch(url *url.URL, origin *url.URL) bool {
	return origin.Host == url.Host && strings.HasPrefix(url.Path, origin.Path)
}

func isSubdomainMatch(url *url.URL, origin *url.URL) bool {
	if len(origin.Host) < 3 || origin.Host[0:2] != "*." {
		return false
	}

	// Check if "*.example.org" matches "example.org"
	if url.Host == origin.Host[2:] && strings.HasPrefix(url.Path, origin.Path) {
		return true
	}

	// Check if "*.example.org" matches "foo.example.org"
	if strings.HasSuffix(url.Host, origin.Host[1:]) && strings.HasPrefix(url.Path, origin.Path) {
		return true
	}

	return false
}

func init() {
	RegisterSource(ImageSourceTypeHTTP, NewHTTPImageSource)
}
