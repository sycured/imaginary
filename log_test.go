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

// testWriter is a simple writer that stores the output.
type testWriter struct {
	buf []byte
}

func (tw *testWriter) Write(b []byte) (int, error) {
	// In this simple case we simply save the latest write.
	tw.buf = b
	return len(b), nil
}

// setupTest creates the test server with the provided log level and returns a pointer to testWriter.
func setupTest(t *testing.T, level string) (*httptest.Server, *testWriter) {
	writer := &testWriter{}
	noopHandler := func(w http.ResponseWriter, r *http.Request) {
		// noopHandler is an intentionally empty handler.
		// It acts as a placeholder for situations where no actual request processing is required.
	}
	// Create a log handler by wrapping the noop handler.
	logHandler := NewLog(http.HandlerFunc(noopHandler), writer, level)
	ts := httptest.NewServer(logHandler)
	// Ensure the server is closed when the test ends.
	t.Cleanup(ts.Close)
	return ts, writer
}

func TestLogInfo(t *testing.T) {
	ts, writer := setupTest(t, "info")
	_, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	data := string(writer.buf)
	if !strings.Contains(data, http.MethodGet) ||
		!strings.Contains(data, "HTTP/1.1") ||
		!strings.Contains(data, " 200 ") {
		t.Fatalf("Invalid log output: %s", data)
	}
}

func TestLogError(t *testing.T) {
	ts, writer := setupTest(t, "error")
	_, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	data := string(writer.buf)
	if data != "" {
		t.Fatalf("Invalid log output: %s", data)
	}
}
