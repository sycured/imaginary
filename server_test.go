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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/h2non/bimg"
)

const CacheControl = "cache-control"
const EmptyResponseBody = "Empty response body"
const InvalidImageType = "Invalid image type"
const InvalidResponseStatusD = "Invalid response status: %d"
const InvalidResponseStatusS = "Invalid response status: %s"
const LargeImageFileWithExt = "large.jpg"
const LargeImageFileWithPath = "testdata/large.jpg"

// sendRequest creates and sends an HTTP request and returns the status code,
// response headers, and body.
func sendRequest(t *testing.T, method, url, contentType string, body io.Reader) (int, http.Header, []byte) {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("cannot perform request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return res.StatusCode, res.Header, respBody
}

// checkResponse validates the response code and that the body is non-empty.
func checkResponse(t *testing.T, statusCode, expectedStatus int, body []byte, errMsg string) {
	t.Helper()

	if statusCode != expectedStatus {
		t.Fatalf("invalid response status: got %d, expected %d", statusCode, expectedStatus)
	}
	if len(body) == 0 {
		t.Fatalf("%s", errMsg)
	}
}

// readTestFile opens a file from the "testdata" directory.
func readTestFile(file string) io.Reader {
	buf, err := os.Open(path.Join("testdata", file))
	if err != nil {
		panic(fmt.Sprintf("failed to open file %s: %v", file, err))
	}
	return buf
}

// assertImageSize checks that the image byte slice has the specified dimensions.
func assertImageSize(t *testing.T, buf []byte, width, height int) {
	t.Helper()
	size, err := bimg.NewImage(buf).Size()
	if err != nil {
		t.Fatalf("failed to get image size: %v", err)
	}
	if size.Width != width || size.Height != height {
		t.Errorf("invalid image size: got %dx%d, expected %dx%d", size.Width, size.Height, width, height)
	}
}

// runTypeAutoCase executes a test case for the automatic type determination request.
func runTypeAutoCase(t *testing.T, acceptHeader, expected string) {
	t.Helper()

	ts := testServer(controller(Crop))
	defer ts.Close()

	buf := readTestFile(LargeImageFileWithExt)
	url := ts.URL + "?width=300&type=auto"

	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", ImageJPEG)
	if acceptHeader != "" {
		req.Header.Set("Accept", acceptHeader)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("cannot perform request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusS, res.Status)
	}
	if res.Header.Get("Content-Length") == "" {
		t.Fatal("Empty content length response")
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}

	assertImageSize(t, image, 300, 1080)
	if bimg.DetermineImageTypeName(image) != expected {
		t.Fatalf(InvalidImageType)
	}
	if res.Header.Get("Vary") != "Accept" {
		t.Fatal("Vary header not set correctly")
	}
}

func TestIndex(t *testing.T) {
	opts := ServerOptions{PathPrefix: "/", MaxAllowedPixels: 18.0}
	ts := testServer(indexController(opts))
	defer ts.Close()

	status, _, body := sendRequest(t, http.MethodGet, ts.URL, "", nil)
	checkResponse(t, status, 200, body, "Invalid body response")

	if !strings.Contains(string(body), "imaginary") {
		t.Fatalf("Invalid body response: %s", body)
	}
}

func TestCrop(t *testing.T) {
	ts := testServer(controller(Crop))
	defer ts.Close()

	imageReader := readTestFile(LargeImageFileWithExt)
	url := ts.URL + "?width=300"

	status, headers, body := sendRequest(t, http.MethodPost, url, ImageJPEG, imageReader)
	checkResponse(t, status, 200, body, EmptyResponseBody)

	if headers.Get("Content-Length") == "" {
		t.Fatal("Empty content length response")
	}

	assertImageSize(t, body, 300, 1080)

	if bimg.DetermineImageTypeName(body) != "jpeg" {
		t.Fatal(InvalidImageType)
	}
}

func TestResize(t *testing.T) {
	ts := testServer(controller(Resize))
	defer ts.Close()

	imageReader := readTestFile(LargeImageFileWithExt)
	url := ts.URL + "?width=300&nocrop=false"

	status, _, body := sendRequest(t, http.MethodPost, url, ImageJPEG, imageReader)
	checkResponse(t, status, 200, body, EmptyResponseBody)

	assertImageSize(t, body, 300, 1080)
	if bimg.DetermineImageTypeName(body) != "jpeg" {
		t.Fatal(InvalidImageType)
	}
}

func TestEnlarge(t *testing.T) {
	ts := testServer(controller(Enlarge))
	defer ts.Close()

	imageReader := readTestFile(LargeImageFileWithExt)
	url := ts.URL + "?width=300&height=200"

	status, _, body := sendRequest(t, http.MethodPost, url, ImageJPEG, imageReader)
	checkResponse(t, status, 200, body, EmptyResponseBody)

	assertImageSize(t, body, 300, 200)
	if bimg.DetermineImageTypeName(body) != "jpeg" {
		t.Fatal(InvalidImageType)
	}
}

func TestExtract(t *testing.T) {
	ts := testServer(controller(Extract))
	defer ts.Close()

	imageReader := readTestFile(LargeImageFileWithExt)
	url := ts.URL + "?top=100&left=100&areawidth=200&areaheight=120"

	status, _, body := sendRequest(t, http.MethodPost, url, ImageJPEG, imageReader)
	checkResponse(t, status, 200, body, EmptyResponseBody)

	assertImageSize(t, body, 200, 120)
	if bimg.DetermineImageTypeName(body) != "jpeg" {
		t.Fatal(InvalidImageType)
	}
}

func TestTypeAuto(t *testing.T) {
	cases := []struct {
		acceptHeader string
		expected     string
	}{
		{"", "jpeg"},
		{"image/webp,*/*", "webp"},
		{"image/png,*/*", "png"},
		{"image/webp;q=0.8,image/jpeg", "webp"},
		{"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8", "webp"},
	}

	for _, testCase := range cases {
		runTypeAutoCase(t, testCase.acceptHeader, testCase.expected)
	}
}

func TestFit(t *testing.T) {
	imageReader := readTestFile(LargeImageFileWithExt)
	original, err := io.ReadAll(imageReader)
	if err != nil {
		t.Fatalf("failed to read image: %v", err)
	}

	// Verify reference image dimensions.
	if err := func() error {
		size, err := bimg.NewImage(original).Size()
		if err != nil {
			return err
		}
		if size.Width != 1920 || size.Height != 1080 {
			return fmt.Errorf("reference image does not have expected dimensions: got %dx%d, expected 1920x1080", size.Width, size.Height)
		}
		return nil
	}(); err != nil {
		t.Errorf("Reference image expectations weren't met: %v", err)
	}

	ts := testServer(controller(Fit))
	defer ts.Close()

	url := ts.URL + "?width=300&height=300"
	status, _, body := sendRequest(t, http.MethodPost, url, ImageJPEG, bytes.NewReader(original))
	checkResponse(t, status, 200, body, EmptyResponseBody)

	// Expected height computed from a ratio of 1.778: 300 x 168.75 rounded to 169
	assertImageSize(t, body, 300, 169)
	if bimg.DetermineImageTypeName(body) != "jpeg" {
		t.Fatal(InvalidImageType)
	}
}

func TestRemoteHTTPSource(t *testing.T) {
	opts := ServerOptions{EnableURLSource: true, MaxAllowedPixels: 18.0}
	fn := ImageMiddleware(opts)(Crop)
	LoadSources(opts)

	tsImage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		buf, _ := os.ReadFile(LargeImageFileWithPath)
		_, _ = w.Write(buf)
	}))
	defer tsImage.Close()

	ts := httptest.NewServer(fn)
	defer ts.Close()

	url := ts.URL + "?width=200&height=200&url=" + tsImage.URL
	status, _, body := sendRequest(t, http.MethodGet, url, "", nil)
	if status != 200 {
		t.Fatalf(InvalidResponseStatusD, status)
	}
	checkResponse(t, status, 200, body, EmptyResponseBody)

	assertImageSize(t, body, 200, 200)
	if bimg.DetermineImageTypeName(body) != "jpeg" {
		t.Fatal(InvalidImageType)
	}
}

func TestInvalidRemoteHTTPSource(t *testing.T) {
	opts := ServerOptions{EnableURLSource: true, MaxAllowedPixels: 18.0}
	fn := ImageMiddleware(opts)(Crop)
	LoadSources(opts)

	tsImage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(400)
	}))
	defer tsImage.Close()

	ts := httptest.NewServer(fn)
	defer ts.Close()

	url := ts.URL + "?width=200&height=200&url=" + tsImage.URL
	status, _, _ := sendRequest(t, http.MethodGet, url, "", nil)
	if status != 400 {
		t.Fatalf(InvalidResponseStatusD, status)
	}
}

func TestMountDirectory(t *testing.T) {
	opts := ServerOptions{Mount: "testdata", MaxAllowedPixels: 18.0}
	fn := ImageMiddleware(opts)(Crop)
	LoadSources(opts)

	ts := httptest.NewServer(fn)
	defer ts.Close()

	url := ts.URL + "?width=200&height=200&file=large.jpg"
	status, _, body := sendRequest(t, http.MethodGet, url, "", nil)
	if status != 200 {
		t.Fatalf(InvalidResponseStatusD, status)
	}
	checkResponse(t, status, 200, body, EmptyResponseBody)

	assertImageSize(t, body, 200, 200)
	if bimg.DetermineImageTypeName(body) != "jpeg" {
		t.Fatal(InvalidImageType)
	}
}

func TestMountInvalidDirectory(t *testing.T) {
	fn := ImageMiddleware(ServerOptions{Mount: "_invalid_", MaxAllowedPixels: 18.0})(Crop)
	ts := httptest.NewServer(fn)
	defer ts.Close()

	url := ts.URL + "?top=100&left=100&areawidth=200&areaheight=120&file=large.jpg"
	status, _, _ := sendRequest(t, http.MethodGet, url, "", nil)
	if status != 400 {
		t.Fatalf(InvalidResponseStatusD, status)
	}
}

func TestMountInvalidPath(t *testing.T) {
	fn := ImageMiddleware(ServerOptions{Mount: "_invalid_"})(Crop)
	ts := httptest.NewServer(fn)
	defer ts.Close()

	url := ts.URL + "?top=100&left=100&areawidth=200&areaheight=120&file=../../large.jpg"
	status, headers, body := sendRequest(t, http.MethodGet, url, "", nil)
	if status != 400 {
		t.Fatalf(InvalidResponseStatusS, headers.Get("Status"))
	}
	// Even if the body is not important here, we check for empty body below.
	checkResponse(t, status, 400, body, EmptyResponseBody)
}

func TestSrcResponseHeaderWithCacheControl(t *testing.T) {
	ttl := 100
	opts := ServerOptions{
		EnableURLSource:    true,
		SrcResponseHeaders: []string{CacheControl, "X-Yep"},
		HTTPCacheTTL:       ttl,
		MaxAllowedPixels:   18.0,
	}
	LoadSources(opts)
	srcHeaderValue := "original-header"

	tsImage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(CacheControl, srcHeaderValue)
		w.Header().Set("X-yep", srcHeaderValue)
		w.Header().Set("X-Nope", srcHeaderValue)
		buf, _ := os.ReadFile(LargeImageFileWithPath)
		_, _ = w.Write(buf)
	}))
	defer tsImage.Close()

	// Use a sub-path to ensure HTTPCacheTTL logic works correctly.
	mux := http.NewServeMux()
	mux.Handle("/foo/", ImageMiddleware(opts)(Resize))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	url := ts.URL + "/foo?width=200&url=" + tsImage.URL
	status, headers, body := sendRequest(t, http.MethodGet, url, "", nil)
	if status != 200 {
		t.Fatalf(InvalidResponseStatusD, status)
	}
	checkResponse(t, status, 200, body, EmptyResponseBody)

	// Check that the specified headers are passed through.
	if headers.Get(CacheControl) != srcHeaderValue || headers.Get("x-yep") != srcHeaderValue {
		t.Fatalf("Header response not passed through properly")
	}
	// Ensure unspecified headers are dropped.
	if headers.Get("x-nope") == srcHeaderValue {
		t.Fatalf("Header response passed through and should not be")
	}
}

func TestSrcResponseHeaderWithoutSrcCacheControl(t *testing.T) {
	ttl := 1234567
	opts := ServerOptions{
		EnableURLSource:    true,
		SrcResponseHeaders: []string{CacheControl, "X-Yep"},
		HTTPCacheTTL:       ttl,
		MaxAllowedPixels:   18.0,
	}
	LoadSources(opts)
	srcHeaderValue := "original-header"

	tsImage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-yep", srcHeaderValue)
		buf, _ := os.ReadFile(LargeImageFileWithPath)
		_, _ = w.Write(buf)
	}))
	defer tsImage.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo/", ImageMiddleware(opts)(Resize))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	url := ts.URL + "/foo?width=200&url=" + tsImage.URL
	status, headers, body := sendRequest(t, http.MethodGet, url, "", nil)
	if status != 200 {
		t.Fatalf(InvalidResponseStatusD, status)
	}
	checkResponse(t, status, 200, body, EmptyResponseBody)

	// The cache-control header should contain the provided TTL value.
	if !strings.Contains(headers.Get(CacheControl), strconv.Itoa(ttl)) {
		t.Fatalf("cache-control header doesn't contain expected value")
	}
}

func controller(op Operation) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		imageHandler(w, r, buf, op, ServerOptions{MaxAllowedPixels: 18.0})
	}
}

func testServer(fn func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(fn))
}

func readFile(file string) io.Reader {
	buf, _ := os.Open(path.Join("testdata", file))
	return buf
}

func assertSize(buf []byte, width, height int) error {
	size, err := bimg.NewImage(buf).Size()
	if err != nil {
		return err
	}
	if size.Width != width || size.Height != height {
		return fmt.Errorf("invalid image size: %dx%d, expected: %dx%d", size.Width, size.Height, width, height)
	}
	return nil
}
