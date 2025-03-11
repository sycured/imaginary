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
const CannotPerformRequest = "Cannot perform the request"
const EmptyResponseBody = "Empty response body"
const InvalidImageType = "Invalid image type"
const InvalidResponseStatusD = "Invalid response status: %d"
const InvalidResponseStatusS = "Invalid response status: %s"
const LargeImageFileWithExt = "large.jpg"
const LargeImageFileWithPath = "testdata/large.jpg"

func TestIndex(t *testing.T) {
	opts := ServerOptions{PathPrefix: "/", MaxAllowedPixels: 18.0}
	ts := testServer(indexController(opts))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusS, res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(body), "imaginary") == false {
		t.Fatalf("Invalid body response: %s", body)
	}
}

func TestCrop(t *testing.T) {
	ts := testServer(controller(Crop))
	buf := readFile(LargeImageFileWithExt)
	url := ts.URL + "?width=300"
	defer ts.Close()

	res, err := http.Post(url, "image/jpeg", buf)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}

	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusS, res.Status)
	}

	if res.Header.Get("Content-Length") == "" {
		t.Fatal("Empty content length response")
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}

	err = assertSize(image, 300, 1080)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(image) != "jpeg" {
		t.Fatalf(InvalidImageType)
	}
}

func TestResize(t *testing.T) {
	ts := testServer(controller(Resize))
	buf := readFile(LargeImageFileWithExt)
	url := ts.URL + "?width=300&nocrop=false"
	defer ts.Close()

	res, err := http.Post(url, "image/jpeg", buf)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}

	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusS, res.Status)
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}

	err = assertSize(image, 300, 1080)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(image) != "jpeg" {
		t.Fatalf(InvalidImageType)
	}
}

func TestEnlarge(t *testing.T) {
	ts := testServer(controller(Enlarge))
	buf := readFile(LargeImageFileWithExt)
	url := ts.URL + "?width=300&height=200"
	defer ts.Close()

	res, err := http.Post(url, "image/jpeg", buf)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}

	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusS, res.Status)
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}

	err = assertSize(image, 300, 200)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(image) != "jpeg" {
		t.Fatalf(InvalidImageType)
	}
}

func TestExtract(t *testing.T) {
	ts := testServer(controller(Extract))
	buf := readFile(LargeImageFileWithExt)
	url := ts.URL + "?top=100&left=100&areawidth=200&areaheight=120"
	defer ts.Close()

	res, err := http.Post(url, "image/jpeg", buf)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}

	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusS, res.Status)
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}

	err = assertSize(image, 200, 120)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(image) != "jpeg" {
		t.Fatalf(InvalidImageType)
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
		{"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8", "webp"}, // Chrome
	}

	for _, test := range cases {
		ts := testServer(controller(Crop))
		buf := readFile(LargeImageFileWithExt)
		url := ts.URL + "?width=300&type=auto"
		defer ts.Close()

		req, _ := http.NewRequest(http.MethodPost, url, buf)
		req.Header.Add("Content-Type", "image/jpeg")
		req.Header.Add("Accept", test.acceptHeader)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(CannotPerformRequest)
		}

		if res.StatusCode != 200 {
			t.Fatalf(InvalidResponseStatusS, res.Status)
		}

		if res.Header.Get("Content-Length") == "" {
			t.Fatal("Empty content length response")
		}

		image, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(image) == 0 {
			t.Fatalf(EmptyResponseBody)
		}

		err = assertSize(image, 300, 1080)
		if err != nil {
			t.Error(err)
		}

		if bimg.DetermineImageTypeName(image) != test.expected {
			t.Fatalf(InvalidImageType)
		}

		if res.Header.Get("Vary") != "Accept" {
			t.Fatal("Vary header not set correctly")
		}
	}
}

func TestFit(t *testing.T) {
	var err error

	buf := readFile(LargeImageFileWithExt)
	original, _ := io.ReadAll(buf)
	err = assertSize(original, 1920, 1080)
	if err != nil {
		t.Errorf("Reference image expecations weren't met")
	}

	ts := testServer(controller(Fit))
	url := ts.URL + "?width=300&height=300"
	defer ts.Close()

	res, err := http.Post(url, "image/jpeg", bytes.NewReader(original))
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}

	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusS, res.Status)
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}

	// The reference image has a ratio of 1.778, this should produce a height of 168.75
	err = assertSize(image, 300, 169)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(image) != "jpeg" {
		t.Fatalf(InvalidImageType)
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
	url := ts.URL + "?width=200&height=200&url=" + tsImage.URL
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}
	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusD, res.StatusCode)
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}

	err = assertSize(image, 200, 200)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(image) != "jpeg" {
		t.Fatalf(InvalidImageType)
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
	url := ts.URL + "?width=200&height=200&url=" + tsImage.URL
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal("Request failed")
	}
	if res.StatusCode != 400 {
		t.Fatalf(InvalidResponseStatusD, res.StatusCode)
	}
}

func TestMountDirectory(t *testing.T) {
	opts := ServerOptions{Mount: "testdata", MaxAllowedPixels: 18.0}
	fn := ImageMiddleware(opts)(Crop)
	LoadSources(opts)

	ts := httptest.NewServer(fn)
	url := ts.URL + "?width=200&height=200&file=large.jpg"
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}
	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusD, res.StatusCode)
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}

	err = assertSize(image, 200, 200)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(image) != "jpeg" {
		t.Fatalf(InvalidImageType)
	}
}

func TestMountInvalidDirectory(t *testing.T) {
	fn := ImageMiddleware(ServerOptions{Mount: "_invalid_", MaxAllowedPixels: 18.0})(Crop)
	ts := httptest.NewServer(fn)
	url := ts.URL + "?top=100&left=100&areawidth=200&areaheight=120&file=large.jpg"
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}

	if res.StatusCode != 400 {
		t.Fatalf(InvalidResponseStatusD, res.StatusCode)
	}
}

func TestMountInvalidPath(t *testing.T) {
	fn := ImageMiddleware(ServerOptions{Mount: "_invalid_"})(Crop)
	ts := httptest.NewServer(fn)
	url := ts.URL + "?top=100&left=100&areawidth=200&areaheight=120&file=../../large.jpg"
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}

	if res.StatusCode != 400 {
		t.Fatalf(InvalidResponseStatusS, res.Status)
	}
}

func TestSrcResponseHeaderWithCacheControl(t *testing.T) {
	opts := ServerOptions{EnableURLSource: true, SrcResponseHeaders: []string{CacheControl, "X-Yep"}, HTTPCacheTTL: 100, MaxAllowedPixels: 18.0}
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

	// need to put the middleware on a sub-path as "/" is treated as a Public path
	// and HTTPCacheTTL logic skips applying the fallback cache-control header
	mux := http.NewServeMux()
	mux.Handle("/foo/", ImageMiddleware(opts)(Resize))
	ts := httptest.NewServer(mux)
	url := ts.URL + "/foo?width=200&url=" + tsImage.URL
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}
	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusD, res.StatusCode)
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}
	// make sure the proper header values are passed through
	if res.Header.Get(CacheControl) != srcHeaderValue || res.Header.Get("x-yep") != srcHeaderValue {
		t.Fatalf("Header response not passed through properly")
	}
	// make sure unspecified headers are dropped
	if res.Header.Get("x-nope") == srcHeaderValue {
		t.Fatalf("Header response passed through and should not be")
	}

}
func TestSrcResponseHeaderWithoutSrcCacheControl(t *testing.T) {
	ttl := 1234567
	opts := ServerOptions{EnableURLSource: true, SrcResponseHeaders: []string{CacheControl, "X-Yep"}, HTTPCacheTTL: ttl, MaxAllowedPixels: 18.0}
	LoadSources(opts)
	srcHeaderValue := "original-header"

	tsImage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-yep", srcHeaderValue)
		buf, _ := os.ReadFile(LargeImageFileWithPath)
		_, _ = w.Write(buf)
	}))
	defer tsImage.Close()

	// need to put the middleware on a sub-path as "/" is treated as a Public path
	// and HTTPCacheTTL logic skips applying the fallback cache-control header
	mux := http.NewServeMux()
	mux.Handle("/foo/", ImageMiddleware(opts)(Resize))
	ts := httptest.NewServer(mux)
	url := ts.URL + "/foo?width=200&url=" + tsImage.URL
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal(CannotPerformRequest)
	}
	if res.StatusCode != 200 {
		t.Fatalf(InvalidResponseStatusD, res.StatusCode)
	}

	image, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(image) == 0 {
		t.Fatalf(EmptyResponseBody)
	}
	// should defer to the provided HTTPCacheTTL value
	if !strings.Contains(res.Header.Get(CacheControl), strconv.Itoa(ttl)) {
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
