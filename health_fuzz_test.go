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
	"math"
	"testing"
)

func FuzzToMegaBytes(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1024))
	f.Add(uint64(1024 * 1024))
	f.Add(uint64(^uint64(0))) // Max uint64
	f.Add(uint64(1))
	f.Add(uint64(1023))
	f.Add(uint64(1025))
	f.Add(uint64(2048))
	f.Add(uint64(4294967295)) // Max uint32
	f.Fuzz(func(t *testing.T, value uint64) {
		result := toMegaBytes(value)
		t.Logf("Input: %v ; Output: %v", value, result)
	})
}

func FuzzRound(f *testing.F) {
	f.Add(0.0)
	f.Add(1.5)
	f.Add(-1.5)
	f.Add(1e20)
	f.Add(-1e20)
	f.Add(math.NaN())
	f.Add(math.Inf(1))
	f.Add(math.Inf(-1))
	f.Add(-0.0)
	f.Add(2.2250738585072014e-308)
	f.Add(-2.2250738585072014e-308)
	f.Add(3.4028234663852886e+38)
	f.Fuzz(func(t *testing.T, value float64) {
		result := round(value)
		t.Logf("Input: %v ; Output: %v", value, result)
	})
}

func FuzzToFixed(f *testing.F) {
	f.Add(0.0, 0)
	f.Add(1.2345, 2)
	f.Add(-1.2345, 3)
	f.Add(math.NaN(), 1)
	f.Add(math.Inf(1), 1)
	f.Add(math.Inf(-1), 1)
	f.Add(1e308, 2)
	f.Add(-1e308, 2)
	f.Add(1.0, -1)
	f.Add(-1.0, -2)
	f.Add(0.0, 10)
	f.Add(1.23456789, 8)
	f.Add(-1.23456789, 8)
	f.Fuzz(func(t *testing.T, num float64, precision int) {
		result := toFixed(num, precision)
		t.Logf("Input Num: %v ; Input Precision %v ; Output: %v", num, precision, result)
	})
}
