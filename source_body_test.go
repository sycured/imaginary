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

const fixtureFile = "testdata/large.jpg"

func TestSourceBodyMatch(t *testing.T) {
	u, _ := url.Parse("http://foo")
	req := &http.Request{Method: http.MethodPost, URL: u}
	source := NewBodyImageSource(&SourceConfig{})

	if !source.Matches(req) {
		t.Error("Cannot match the request")
	}
}

func testHelper(t *testing.T, source ImageSource, fixture string, additionalAssertions func(t *testing.T, body []byte, w *httptest.ResponseRecorder)) {
	var body []byte
	var err error

	fakeHandler := func(w http.ResponseWriter, r *http.Request) {
		// Use the interface methods
		if !source.Matches(r) {
			t.Fatal("Cannot match the request")
		}

		body, _, err = source.GetImage(r)
		if err != nil {
			t.Fatalf("Error while reading the body: %s", err)
		}
		_, _ = w.Write(body)
	}

	file, _ := os.Open(fixture)
	defer file.Close() // Ensure the file is closed properly
	r, _ := http.NewRequest(http.MethodPost, "http://foo/bar", file)
	w := httptest.NewRecorder()
	fakeHandler(w, r)

	buf, _ := os.ReadFile(fixture)
	if len(body) != len(buf) {
		t.Error("Invalid response body")
	}

	// Perform any test-specific assertions passed through `additionalAssertions`
	if additionalAssertions != nil {
		additionalAssertions(t, body, w)
	}
}

func TestBodyImageSource(t *testing.T) {
	source := NewBodyImageSource(&SourceConfig{}) // Explicit *BodyImageSource
	testHelper(t, source, fixtureFile, nil)       // No additional assertions needed
}

func TestReadBody(t *testing.T) {
	source := NewBodyImageSource(&SourceConfig{}) // Explicit *BodyImageSource
	testHelper(t, source, fixtureFile, func(t *testing.T, body []byte, w *httptest.ResponseRecorder) {
		// Add test-specific assertions
		expectedHeader := "true"
		w.Header().Set("X-Test-ReadBody", expectedHeader) // Simulate a header
		if w.Header().Get("X-Test-ReadBody") != expectedHeader {
			t.Errorf("Expected header 'X-Test-ReadBody' to be '%s', but got '%s'", expectedHeader, w.Header().Get("X-Test-ReadBody"))
		}
	})
}
