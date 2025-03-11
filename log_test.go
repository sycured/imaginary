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
	"strings"
	"testing"
)

type fakeWriter func([]byte) (int, error)

func (fake fakeWriter) Write(buf []byte) (int, error) {
	return fake(buf)
}

func TestLogInfo(t *testing.T) {
	var buf []byte
	writer := fakeWriter(func(b []byte) (int, error) {
		buf = b
		return 0, nil
	})

	noopHandler := func(w http.ResponseWriter, r *http.Request) {}
	log := NewLog(http.HandlerFunc(noopHandler), writer, "info")

	ts := httptest.NewServer(log)
	defer ts.Close()

	_, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	data := string(buf)
	if strings.Contains(data, http.MethodGet) == false ||
		strings.Contains(data, "HTTP/1.1") == false ||
		strings.Contains(data, " 200 ") == false {
		t.Fatalf("Invalid log output: %s", data)
	}
}

func TestLogError(t *testing.T) {
	var buf []byte
	writer := fakeWriter(func(b []byte) (int, error) {
		buf = b
		return 0, nil
	})

	noopHandler := func(w http.ResponseWriter, r *http.Request) {}
	log := NewLog(http.HandlerFunc(noopHandler), writer, "error")

	ts := httptest.NewServer(log)
	defer ts.Close()

	_, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	data := string(buf)
	if data != "" {
		t.Fatalf("Invalid log output: %s", data)
	}
}
