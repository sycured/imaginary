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

func TestBimgOptions(t *testing.T) {
	imgOpts := ImageOptions{
		Width:  500,
		Height: 600,
	}
	opts := BimgOptions(imgOpts)

	if opts.Width != imgOpts.Width || opts.Height != imgOpts.Height {
		t.Error("Invalid width and height")
	}
}
