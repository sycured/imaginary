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
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

func isSuffixLetter(s string) bool {
	if len(s) == 0 {
		return false // Empty string has no suffix
	}
	lastRune := []rune(s)[len(s)-1]
	return unicode.IsLetter(lastRune)
}

func isAlphanumOnly(s string) bool {
	regex := regexp.MustCompile(`[^a-zA-Z0-9 ]`)

	if regex.MatchString(s) {
		return false
	} else {
		return true
	}
}

func correctMemoryValue(value string) bool {
	hasDigit := false
	hasLetter := false
	if !isAlphanumOnly(value) {
		return false
	}

	if !isSuffixLetter(value) {
		return false
	}

	for idx, char := range value {

		if idx == 0 && !unicode.IsDigit(char) {
			return false
		}

		if !unicode.IsDigit(char) && !unicode.IsLetter(char) && char == '"' {
			return false
		}

		if unicode.IsDigit(char) {
			hasDigit = true
		} else if unicode.IsLetter(char) {
			hasLetter = true
		}

		if hasDigit && hasLetter {
			return true
		}
	}

	return false
}

func getUnikernelMemory() (int64, error) {
	const (
		KB int64 = 1024
		MB       = KB * 1024
		GB       = MB * 1024
		TB       = GB * 1024
	)

	suffixes := map[string]int64{
		"KB": KB, "K": KB, "MB": MB, "M": MB, "GB": GB, "G": GB, "TB": TB, "T": TB,
	}

	memStr := strings.TrimSpace(strings.ToUpper(os.Getenv("UNIKERNEL_MEMORY")))
	if memStr == "" || memStr == "0" {
		return 0, fmt.Errorf("UNIKERNEL_MEMORY is empty or 0")
	}

	if !correctMemoryValue(memStr) {
		return 0, fmt.Errorf("UNIKERNEL_MEMORY has an invalid format: %#v", memStr)
	}

	re := regexp.MustCompile(`([\d\.]+)\s*([a-zA-Z]+)`)
	matches := re.FindStringSubmatch(memStr)

	if len(matches) != 3 {
		return 0, fmt.Errorf("FAIL: Regex matches are 3 ... %#v", matches)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, err
	}
	unit := matches[2]

	return value * suffixes[unit], nil
}
