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

func setupTestCases(f *testing.F) {
	testCases := [][4]int{
		{1280, 1000, 710, 555},
		{1279, 1000, 710, 555},
		{900, 500, 312, 173},
		{900, 500, 313, 174},
		{1299, 2000, 649, 999},
		{1500, 2000, 710, 947},
	}
	for _, tc := range testCases {
		f.Add(tc[0], tc[1], tc[2], tc[3])
	}
}

func shouldSkipTest(t *testing.T, imageWidth, imageHeight, fitWidth, fitHeight int) bool {
	if imageWidth < 0 || imageHeight < 0 || fitWidth < 0 || fitHeight < 0 {
		t.Skip("Skipping negative dimensions")
		return true
	}
	if imageWidth > 1000000 || imageHeight > 1000000 || fitWidth > 1000000 || fitHeight > 1000000 {
		t.Skip("Skipping very large dimensions")
		return true
	}
	if imageWidth == 0 || imageHeight == 0 {
		t.Skip("Skipping zero image dimensions")
		return true
	}
	return false
}

func validateDimensions(t *testing.T, imageWidth, imageHeight, fitWidth, fitHeight int) {
	resultWidth, resultHeight := calculateDestinationFitDimension(imageWidth, imageHeight, fitWidth, fitHeight)
	validateResultBounds(t, resultWidth, resultHeight, fitWidth, fitHeight, imageWidth, imageHeight)
	validateAspectRatio(t, resultWidth, resultHeight, imageWidth, imageHeight, fitWidth, fitHeight)
}

func validateResultBounds(t *testing.T, resultWidth, resultHeight, fitWidth, fitHeight, imageWidth, imageHeight int) {
	if resultWidth > fitWidth || resultHeight > fitHeight {
		t.Errorf("Result dimensions (%d, %d) exceed input fit dimensions (%d, %d). Input: iW:%d, iH:%d, fW:%d, fH:%d",
			resultWidth, resultHeight, fitWidth, fitHeight, imageWidth, imageHeight, fitWidth, fitHeight)
	}
}

func validateAspectRatio(t *testing.T, resultWidth, resultHeight, imageWidth, imageHeight, fitWidth, fitHeight int) {
	isConstrainedByWidth := imageWidth*fitHeight > fitWidth*imageHeight
	if isConstrainedByWidth {
		validateWidthConstrained(t, resultWidth, resultHeight, imageWidth, imageHeight, fitWidth, fitHeight)
	} else {
		validateHeightConstrained(t, resultWidth, resultHeight, imageWidth, imageHeight, fitWidth, fitHeight)
	}
}

func validateWidthConstrained(t *testing.T, resultWidth, resultHeight, imageWidth, imageHeight, fitWidth, fitHeight int) {
	if resultWidth != fitWidth {
		t.Errorf("Constrained by width, but resultWidth (%d) != input fitWidth (%d). Input: iW:%d, iH:%d, fW:%d, fH:%d",
			resultWidth, fitWidth, imageWidth, imageHeight, fitWidth, fitHeight)
	}
	expectedHeight := int(math.Round(float64(resultWidth) * float64(imageHeight) / float64(imageWidth)))
	if resultHeight != expectedHeight {
		t.Errorf("Aspect ratio mismatch (width constrained). Expected H: %d, Got H: %d. Input: iW:%d, iH:%d, fW:%d, fH:%d",
			expectedHeight, resultHeight, imageWidth, imageHeight, fitWidth, fitHeight)
	}
}

func validateHeightConstrained(t *testing.T, resultWidth, resultHeight, imageWidth, imageHeight, fitWidth, fitHeight int) {
	if resultHeight != fitHeight {
		t.Errorf("Constrained by height, but resultHeight (%d) != input fitHeight (%d). Input: iW:%d, iH:%d, fW:%d, fH:%d",
			resultHeight, fitHeight, imageWidth, imageHeight, fitWidth, fitHeight)
	}
	expectedWidth := int(math.Round(float64(resultHeight) * float64(imageWidth) / float64(imageHeight)))
	if resultWidth != expectedWidth {
		t.Errorf("Aspect ratio mismatch (height constrained). Expected W: %d, Got W: %d. Input: iW:%d, iH:%d, fW:%d, fH:%d",
			expectedWidth, resultWidth, imageWidth, imageHeight, fitWidth, fitHeight)
	}
}

func FuzzCalculateDestinationFitDimension(f *testing.F) {
	setupTestCases(f)
	f.Fuzz(func(t *testing.T, imageWidth, imageHeight, fitWidth, fitHeight int) {
		if shouldSkipTest(t, imageWidth, imageHeight, fitWidth, fitHeight) {
			return
		}
		validateDimensions(t, imageWidth, imageHeight, fitWidth, fitHeight)
	})
}
