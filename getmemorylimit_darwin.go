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
	"fmt"
	"math"

	"golang.org/x/sys/unix"
)

// getMemoryLimit returns the total physical memory of the host and an error if it fails.
func getMemoryLimit() (int64, error) {
	mem, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0, fmt.Errorf("error retrieving memory using sysctl: %w", err)
	}
	if mem > math.MaxInt64 {
		return 0, fmt.Errorf("memory size exceeds the maximum integer value: reduced at MaxInt64")
	}
	return int64(mem), nil
}
