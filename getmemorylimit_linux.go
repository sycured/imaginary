//go:build linux
// +build linux

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
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// getMemoryLimit returns the total physical memory of the host.
func getMemoryLimit() int64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		log.Printf("Error opening /proc/meminfo: %v", err)
		return 0
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Look for the line that starts with "MemTotal:"
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// The value is provided in kilobytes.
				memKB, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return memKB * 1024
				}
				log.Printf("Error parsing mem value %q: %v", fields[1], err)
				return 0
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error scanning /proc/meminfo: %v", err)
	}

	return 0
}
