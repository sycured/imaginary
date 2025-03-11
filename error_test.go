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

import "testing"

func TestDefaultError(t *testing.T) {
	err := NewError("oops!\n\n", 503)

	if err.Error() != "oops!" {
		t.Fatal("Invalid error message")
	}
	if err.Code != 503 {
		t.Fatal("Invalid error code")
	}

	code := err.HTTPCode()
	if code != 503 {
		t.Fatalf("Invalid HTTP error status: %d", code)
	}

	json := string(err.JSON())
	if json != "{\"message\":\"oops!\",\"status\":503}" {
		t.Fatalf("Invalid JSON output: %s", json)
	}
}
