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
	"os"
	"testing"
)

func TestFileSystemImageSource(t *testing.T) {
	var body []byte
	var err error
	const fixtureFile = "testdata/large.jpg"

	source := NewFileSystemImageSource(&SourceConfig{MountPath: "testdata"})
	fakeHandler := func(w http.ResponseWriter, r *http.Request) {
		if !source.Matches(r) {
			t.Fatal("Cannot match the request")
		}

		body, _, err = source.GetImage(r)
		if err != nil {
			t.Fatalf("Error while reading the body: %s", err)
		}
		_, _ = w.Write(body)
	}

	file, _ := os.Open(fixtureFile)
	r, _ := http.NewRequest(http.MethodGet, "http://foo/bar?file=large.jpg", file)
	w := httptest.NewRecorder()
	fakeHandler(w, r)

	buf, _ := os.ReadFile(fixtureFile)
	if len(body) != len(buf) {
		t.Error("Invalid response body")
	}
}
