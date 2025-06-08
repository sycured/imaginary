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
	"net/url"
	"strings"
	"testing"
)

func FuzzBuildParamsFromQuery(f *testing.F) {
	// Seed corpus with various valid and potentially problematic query strings
	f.Add("width=100&height=50&type=jpeg")    // Valid case
	f.Add("quality=90&force=true")            // Another valid case
	f.Add("")                                 // Empty query
	f.Add("width=abc&height=xyz")             // Invalid integer values
	f.Add("force=maybe")                      // Invalid boolean value
	f.Add("color=255,0,0,extra")              // Invalid color format
	f.Add("gravity=unknown")                  // Invalid gravity value
	f.Add("operations=[{\"name\":\"crop\"}]") // Valid JSON operation
	f.Add("operations={[invalid json")        // Invalid JSON operation
	f.Add("width=100000000000000000000")      // Large number
	f.Add("text=%00%FFinvalid")               // Non-UTF8 characters
	f.Add("a=1&a=2&a=3")                      // Repeated parameters
	f.Add(string(make([]byte, 1024*10)))      // Very long query string

	f.Fuzz(func(t *testing.T, queryString string) {
		queryValues, err := url.ParseQuery(queryString)
		imageOptions, err2 := buildParamsFromQuery(queryValues)
		if err != nil && !strings.Contains(err.Error(), "invalid URL escape") && !strings.Contains(err.Error(), "invalid semicolon separator in query") {
			t.Errorf("Input: %#v ; Output: %v ; Error: %#v", queryValues, imageOptions, err.Error())
		}
		t.Logf("Input: %s, ImageOptions: %v, err: %v", queryString, imageOptions, err2)
	})
}
