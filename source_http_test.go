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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

const CannotMatchRequest = "Cannot match the request"
const ExpectedAllowedOrigins = "Expected '%s' to be allowed with origins: %+v"
const fixtureImage = "testdata/large.jpg"
const fixture1024Bytes = "testdata/1024bytes"
const HttpBarCom = "http://bar.com"
const HttpFooBarUrl = "http://foo/bar?url="
const HttpFooBarUrlBarCom = HttpFooBarUrl + HttpBarCom
const XCustom = "X-Custom"
const XToken = "X-Token"

func TestHttpImageSource(t *testing.T) {
	var body []byte
	var err error

	buf, _ := os.ReadFile(fixtureImage)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buf)
	}))
	defer ts.Close()

	source := NewHTTPImageSource(&SourceConfig{})
	fakeHandler := func(w http.ResponseWriter, r *http.Request) {
		if !source.Matches(r) {
			t.Fatal(CannotMatchRequest)
		}

		body, _, err = source.GetImage(r)
		if err != nil {
			t.Fatalf("Error while reading the body: %s", err)
		}
		_, _ = w.Write(body)
	}

	r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrl+ts.URL, nil)
	w := httptest.NewRecorder()
	fakeHandler(w, r)

	if len(body) != len(buf) {
		t.Error("Invalid response body")
	}
}

func TestHttpImageSourceAllowedOrigin(t *testing.T) {
	buf, _ := os.ReadFile(fixtureImage)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buf)
	}))
	defer ts.Close()

	origin, _ := url.Parse(ts.URL)
	origins := []*url.URL{origin}
	source := NewHTTPImageSource(&SourceConfig{AllowedOrigins: origins})

	fakeHandler := func(w http.ResponseWriter, r *http.Request) {
		if !source.Matches(r) {
			t.Fatal(CannotMatchRequest)
		}

		body, _, err := source.GetImage(r)
		if err != nil {
			t.Fatalf("Error while reading the body: %s", err)
		}
		_, _ = w.Write(body)

		if len(body) != len(buf) {
			t.Error("Invalid response body length")
		}
	}

	r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrl+ts.URL, nil)
	w := httptest.NewRecorder()
	fakeHandler(w, r)
}

func TestHttpImageSourceNotAllowedOrigin(t *testing.T) {
	origin, _ := url.Parse("http://foo")
	origins := []*url.URL{origin}
	source := NewHTTPImageSource(&SourceConfig{AllowedOrigins: origins})

	fakeHandler := func(w http.ResponseWriter, r *http.Request) {
		if !source.Matches(r) {
			t.Fatal(CannotMatchRequest)
		}

		_, _, err := source.GetImage(r)
		if err == nil {
			t.Fatal("Error cannot be empty")
		}

		if err.Error() != "not allowed remote URL origin: bar.com" {
			t.Fatalf("Invalid error message: %s", err)
		}
	}

	r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrlBarCom, nil)
	w := httptest.NewRecorder()
	fakeHandler(w, r)
}

func TestHttpImageSourceForwardAuthHeader(t *testing.T) {
	cases := []string{
		"X-Forward-Authorization",
		"Authorization",
	}

	for _, header := range cases {
		r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrlBarCom, nil)
		r.Header.Set(header, "foobar")

		source := &HTTPImageSource{&SourceConfig{AuthForwarding: true}}
		if !source.Matches(r) {
			t.Fatal(CannotMatchRequest)
		}

		oreq := &http.Request{Header: make(http.Header)}
		source.setAuthorizationHeader(oreq, r)

		if oreq.Header.Get("Authorization") != "foobar" {
			t.Fatal("Mismatch Authorization header")
		}
	}
}

func TestHttpImageSourceForwardHeaders(t *testing.T) {
	cases := []string{
		XCustom,
		XToken,
	}

	for _, header := range cases {
		r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrlBarCom, nil)
		r.Header.Set(header, "foobar")

		source := &HTTPImageSource{&SourceConfig{ForwardHeaders: cases}}
		if !source.Matches(r) {
			t.Fatal(CannotMatchRequest)
		}

		oreq := &http.Request{Header: make(http.Header)}
		source.setForwardHeaders(oreq, r)

		if oreq.Header.Get(header) != "foobar" {
			t.Fatal("Mismatch custom header")
		}
	}
}

func TestHttpImageSourceNotForwardHeaders(t *testing.T) {
	cases := []string{
		XCustom,
		XToken,
	}

	testURL := createURL(HttpBarCom, t)

	r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrl+testURL.String(), nil)
	r.Header.Set("Not-Forward", "foobar")

	source := &HTTPImageSource{&SourceConfig{ForwardHeaders: cases}}
	if !source.Matches(r) {
		t.Fatal(CannotMatchRequest)
	}

	oreq := newHTTPRequest(source, r, http.MethodGet, testURL)

	if oreq.Header.Get("Not-Forward") != "" {
		t.Fatal("Forwarded unspecified header")
	}
}

func TestHttpImageSourceForwardedHeadersNotOverride(t *testing.T) {
	cases := []string{
		"Authorization",
		XCustom,
	}

	testURL := createURL(HttpBarCom, t)

	r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrl+testURL.String(), nil)
	r.Header.Set("Authorization", "foobar")

	source := &HTTPImageSource{&SourceConfig{Authorization: "ValidAPIKey", ForwardHeaders: cases}}
	if !source.Matches(r) {
		t.Fatal(CannotMatchRequest)
	}

	oreq := newHTTPRequest(source, r, http.MethodGet, testURL)

	if oreq.Header.Get("Authorization") != "ValidAPIKey" {
		t.Fatal("Authorization header override")
	}
}

func TestHttpImageSourceCaseSensitivityInForwardedHeaders(t *testing.T) {
	cases := []string{
		XCustom,
		XToken,
	}

	testURL := createURL(HttpBarCom, t)

	r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrl+testURL.String(), nil)
	r.Header.Set(XCustom, "foobar")

	source := &HTTPImageSource{&SourceConfig{ForwardHeaders: cases}}
	if !source.Matches(r) {
		t.Fatal(CannotMatchRequest)
	}

	oreq := newHTTPRequest(source, r, http.MethodGet, testURL)

	if oreq.Header.Get(XCustom) == "" {
		t.Fatal("Case sensitive not working on forwarded headers")
	}
}

func TestHttpImageSourceEmptyForwardedHeaders(t *testing.T) {
	var cases []string

	testURL := createURL(HttpBarCom, t)

	r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrl+testURL.String(), nil)

	source := &HTTPImageSource{&SourceConfig{ForwardHeaders: cases}}
	if !source.Matches(r) {
		t.Fatal(CannotMatchRequest)
	}

	if len(source.Config.ForwardHeaders) != 0 {
		t.Log(source.Config.ForwardHeaders)
		t.Fatal("Set empty custom header")
	}

	oreq := newHTTPRequest(source, r, http.MethodGet, testURL)

	if oreq == nil {
		t.Fatal("Error creating request using empty custom headers")
	}
}

func TestHttpImageSourceError(t *testing.T) {
	var err error

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte("Not found"))
	}))
	defer ts.Close()

	source := NewHTTPImageSource(&SourceConfig{})
	fakeHandler := func(w http.ResponseWriter, r *http.Request) {
		if !source.Matches(r) {
			t.Fatal(CannotMatchRequest)
		}

		_, _, err = source.GetImage(r)
		if err == nil {
			t.Fatalf("Server response should not be valid: %s", err)
		}
	}

	r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrl+ts.URL, nil)
	w := httptest.NewRecorder()
	fakeHandler(w, r)
}

func TestHttpImageSourceExceedsMaximumAllowedLength(t *testing.T) {
	var body []byte
	var err error

	buf, _ := os.ReadFile(fixture1024Bytes)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buf)
	}))
	defer ts.Close()

	source := NewHTTPImageSource(&SourceConfig{
		MaxAllowedSize: 1023,
	})
	fakeHandler := func(w http.ResponseWriter, r *http.Request) {
		if !source.Matches(r) {
			t.Fatal(CannotMatchRequest)
		}

		body, _, err = source.GetImage(r)
		if err == nil {
			t.Fatalf("It should not allow a request to image exceeding maximum allowed size: %s", err)
		}
		_, _ = w.Write(body)
	}

	r, _ := http.NewRequest(http.MethodGet, HttpFooBarUrl+ts.URL, nil)
	w := httptest.NewRecorder()
	fakeHandler(w, r)
}

func TestShouldRestrictOrigin(t *testing.T) {
	// Prepare the various origins
	plainOrigins := parseOrigins("https://example.org")
	wildCardOrigins := parseOrigins("https://localhost,https://*.example.org,https://some.s3.bucket.on.aws.org,https://*.s3.bucket.on.aws.org")
	withPathOrigins := parseOrigins("https://localhost/foo/bar/,https://*.example.org/foo/,https://some.s3.bucket.on.aws.org/my/bucket/,https://*.s3.bucket.on.aws.org/my/bucket/,https://no-leading-path-slash.example.org/assets")
	with2Buckets := parseOrigins("https://some.s3.bucket.on.aws.org/my/bucket1/,https://some.s3.bucket.on.aws.org/my/bucket2/")
	pathWildCard := parseOrigins("https://some.s3.bucket.on.aws.org/my-bucket-name*")

	// Define a table of test cases.
	tests := []struct {
		name               string
		urlStr             string
		origins            []*url.URL
		restrictedExpected bool // true means origin should be restricted
	}{
		{
			name:               "Plain origin",
			urlStr:             "https://example.org/logo.jpg",
			origins:            plainOrigins,
			restrictedExpected: false,
		},
		{
			name:               "Wildcard origin, plain URL",
			urlStr:             "https://example.org/logo.jpg",
			origins:            wildCardOrigins,
			restrictedExpected: false,
		},
		{
			name:               "Wildcard origin, sub domain URL",
			urlStr:             "https://node-42.example.org/logo.jpg",
			origins:            wildCardOrigins,
			restrictedExpected: false,
		},
		{
			name:               "Wildcard origin, sub-sub domain URL",
			urlStr:             "https://n.s3.bucket.on.aws.org/our/bucket/logo.jpg",
			origins:            wildCardOrigins,
			restrictedExpected: false,
		},
		{
			name:               "Incorrect domain URL (plain origins)",
			urlStr:             "https://myexample.org/logo.jpg",
			origins:            plainOrigins,
			restrictedExpected: true,
		},
		{
			name:               "Incorrect domain URL (wildcard origins)",
			urlStr:             "https://myexample.org/logo.jpg",
			origins:            wildCardOrigins,
			restrictedExpected: true,
		},
		{
			name:               "Loopback origin with path, correct URL",
			urlStr:             "https://localhost/foo/bar/logo.png",
			origins:            withPathOrigins,
			restrictedExpected: false,
		},
		{
			name:               "Wildcard origin with path, correct URL",
			urlStr:             "https://our.company.s3.bucket.on.aws.org/my/bucket/logo.gif",
			origins:            withPathOrigins,
			restrictedExpected: false,
		},
		{
			name:               "Wildcard origin with partial path, correct URL",
			urlStr:             "https://our.company.s3.bucket.on.aws.org/my/bucket/a/b/c/d/e/logo.gif",
			origins:            withPathOrigins,
			restrictedExpected: false,
		},
		{
			name:               "Wildcard origin with partial path, correct URL double slashes",
			urlStr:             "https://static.example.org/foo//a//b//c/d/e/logo.webp",
			origins:            withPathOrigins,
			restrictedExpected: false,
		},
		{
			name:               "Wildcard origin with path missing trailing slash",
			urlStr:             "https://no-leading-path-slash.example.org/assets/logo.webp",
			origins:            parseOrigins("https://*.example.org/assets"),
			restrictedExpected: false,
		},
		{
			name:               "Loopback origin with path, incorrect URL",
			urlStr:             "https://localhost/wrong/logo.png",
			origins:            withPathOrigins,
			restrictedExpected: true,
		},
		{
			name:               "2 buckets, bucket1",
			urlStr:             "https://some.s3.bucket.on.aws.org/my/bucket1/logo.jpg",
			origins:            with2Buckets,
			restrictedExpected: false,
		},
		{
			name:               "2 buckets, bucket2",
			urlStr:             "https://some.s3.bucket.on.aws.org/my/bucket2/logo.jpg",
			origins:            with2Buckets,
			restrictedExpected: false,
		},
		{
			name:               "Path wildcard, allowed",
			urlStr:             "https://some.s3.bucket.on.aws.org/my-bucket-name/logo.jpg",
			origins:            pathWildCard,
			restrictedExpected: false,
		},
		{
			name:               "Path wildcard, restricted",
			urlStr:             "https://some.s3.bucket.on.aws.org/my-other-bucket-name/logo.jpg",
			origins:            pathWildCard,
			restrictedExpected: true,
		},
	}

	// Iterate over the test cases.
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testURL := createURL(tc.urlStr, t)
			actual := shouldRestrictOrigin(testURL, tc.origins)
			if actual != tc.restrictedExpected {
				if tc.restrictedExpected {
					t.Errorf("Expected '%s' to be restricted with origins: %+v", testURL, tc.origins)
				} else {
					t.Errorf(ExpectedAllowedOrigins, testURL, tc.origins)
				}
			}
		})
	}
}

func TestParseOrigins(t *testing.T) {
	t.Run("Appending a trailing slash on paths", func(t *testing.T) {
		origins := parseOrigins("http://foo.example.org/assets")
		if origins[0].Path != "/assets/" {
			t.Errorf("Expected the path to have a trailing /, instead it was: %q", origins[0].Path)
		}
	})

	t.Run("Paths should not receive multiple trailing slashes", func(t *testing.T) {
		origins := parseOrigins("http://foo.example.org/assets/")
		if origins[0].Path != "/assets/" {
			t.Errorf("Expected the path to have a single trailing /, instead it was: %q", origins[0].Path)
		}
	})

	t.Run("Empty paths are fine", func(t *testing.T) {
		origins := parseOrigins("http://foo.example.org")
		if origins[0].Path != "" {
			t.Errorf("Expected the path to remain empty, instead it was: %q", origins[0].Path)
		}
	})

	t.Run("Root paths are fine", func(t *testing.T) {
		origins := parseOrigins("http://foo.example.org/")
		if origins[0].Path != "/" {
			t.Errorf("Expected the path to remain a slash, instead it was: %q", origins[0].Path)
		}
	})
}

func createURL(urlStr string, t *testing.T) *url.URL {
	t.Helper()

	result, err := url.Parse(urlStr)

	if err != nil {
		t.Error("Test setup failed, unable to parse test URL")
	}

	return result
}
