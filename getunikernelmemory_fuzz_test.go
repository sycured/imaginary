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
	"strings"
	"testing"
)

func FuzzGetUnikernelMemory(f *testing.F) {
	f.Add("1K")
	f.Add("256 KB")
	f.Add("100 M")
	f.Add("8MB")
	f.Add("1G")
	f.Add("2 GB")
	f.Add("5T")
	f.Add("20 TB")
	f.Fuzz(func(t *testing.T, value string) {
		_ = os.Setenv("UNIKERNEL_MEMORY", value)
		result, err := getUnikernelMemory()
		if err != nil {
			if strings.HasPrefix(err.Error(), "FAIL") {
				t.Errorf("Input: %v ; Output: %v ; Error: %v", value, result, err.Error())
			}
		}
		t.Logf("Input: %v ; Output: %v", value, result)
	})
}
