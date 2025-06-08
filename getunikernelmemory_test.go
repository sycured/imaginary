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
	"os"
	"testing"
)

const InvalidParam = "Invalid param: %#v != %d ; Error: %#v"

func TestGetUnikernelMemory(t *testing.T) {
	tests := []struct {
		value    string
		expected uint64
	}{
		{"", 0},
		{"0", 0},
		{" ", 0},
		{"L9", 0},
		{"1K", 1024},
		{"256 KB", 262144},
		{"100 M", 104857600},
		{"8MB", 8388608},
		{"1G", 1073741824},
		{"2 GB", 2147483648},
		{"5T", 5497558138880},
		{"20 TB", 21990232555520},
	}

	for _, test := range tests {
		_ = os.Setenv("UNIKERNEL_MEMORY", test.value)
		val, err := getUnikernelMemory()
		if val != int64(test.expected) {
			t.Errorf(InvalidParam, val, test.expected, err)
		}
	}
}
