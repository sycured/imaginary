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
	"mime/multipart"
	"os"
	"strconv"
	"strings"
)

// getMemoryLimit returns the memory limit enforced by cgroups if available.
// It checks the file "/sys/fs/cgroup/memory.max" and, if a valid numeric value is found,
// returns it as bytes. If the value is "max" (indicating no limit) or any error occurs,
// it falls back to retrieving the physical memory of the host.
func getMemoryLimit() int64 {
	const cgroupMemoryMax = "/sys/fs/cgroup/memory.max"

	data, err := os.ReadFile(cgroupMemoryMax)
	if err == nil {
		val := strings.TrimSpace(string(data))
		// "max" indicates no limit has been enforced.
		if val != "max" {
			memBytes, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				return memBytes
			}
			log.Printf("Error parsing %q from %s: %v", val, cgroupMemoryMax, err)
		}
	} else {
		log.Printf("Error reading %s: %v", cgroupMemoryMax, err)
	}

	// Fallback: return the total physical memory of the host.
	return getPhysicalMemoryLimit()
}

// getPhysicalMemoryLimit returns the total physical memory of the host in bytes
// by reading the "/proc/meminfo" file.
func getPhysicalMemoryLimit() int64 {
	const procMeminfo = "/proc/meminfo"
	file, err := os.Open(procMeminfo)
	if err != nil {
		log.Printf("Error opening %s: %v", procMeminfo, err)
		return 0
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
					return memKB * 1024
				}
				log.Printf("Error parsing memory value %q: %v", fields[1], err)
				return 0
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error scanning %s: %v", procMeminfo, err)
	}

	return 0
}
