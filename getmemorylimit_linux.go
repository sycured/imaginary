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
	"mime/multipart"
	"os"
	"strconv"
	"strings"
)

// getMemoryLimit returns the memory limit enforced by cgroups if available,
// or falls back to retrieving the physical memory of the host if no limit is found.
func getMemoryLimit() (int64, error) {
	const cgroupMemoryMax = "/sys/fs/cgroup/memory.max"

	data, err := os.ReadFile(cgroupMemoryMax)
	if err == nil {
		val := strings.TrimSpace(string(data))
		// "max" indicates no limit has been enforced.
		if val != "max" {
			memBytes, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				return int64(memBytes), nil
			}
			fmt.Errorf("Error reading %s: %v", cgroupMemoryMax, err)
		}
	} else {
		// Fallback: retrieve the total physical memory of the host.
		memLimit, err := getPhysicalMemoryLimit()
		if err == nil {
			return memLimit, nil
		}
		return 0, fmt.Errorf("Failed to determine memory limit")
	}
}

// getPhysicalMemoryLimit returns the total physical memory of the host in bytes
// by reading the "/proc/meminfo" file and handling potential errors.
func getPhysicalMemoryLimit() (int64, error) {
	const procMeminfo = "/proc/meminfo"
	file, err := os.Open(procMeminfo)
	if err != nil {
		return 0, fmt.Errorf("Error opening %s: %v", procMeminfo, err)
	}

	defer func(file multipart.File) {
		_ = file.Close()
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				memKB, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return memKB * 1024, nil
				}
				log.Printf("Error parsing memory value %q: %v", fields[1],
					err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("Error scanning %s: %v", procMeminfo, err)
	}

	return 0, nil
}
