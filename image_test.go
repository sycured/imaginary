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
	"io"
	"testing"
)

const CannotProcessImageS = "Cannot process image: %s"
const ImaginaryJpeg = "imaginary.jpg"
const InvalidImageSize = "Invalid image size, expected: %dx%d"
const InvalidMimeType = "Invalid image MIME type"

func TestImageResize(t *testing.T) {
	tests := []struct {
		name           string
		opts           ImageOptions
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "Width and Height defined",
			opts:           ImageOptions{Width: 300, Height: 300},
			expectedWidth:  300,
			expectedHeight: 300,
		},
		{
			name:           "Width defined",
			opts:           ImageOptions{Width: 300},
			expectedWidth:  300,
			expectedHeight: 404,
		},
		{
			name:          "Width defined with NoCrop=false",
			opts:          ImageOptions{Width: 300, NoCrop: false, IsDefinedField: IsDefinedField{NoCrop: true}},
			expectedWidth: 300,
			// The original image is 550x740; NoCrop=false forces full height of 740
			expectedHeight: 740,
		},
		{
			name:           "Width defined with NoCrop=true",
			opts:           ImageOptions{Width: 300, NoCrop: true, IsDefinedField: IsDefinedField{NoCrop: true}},
			expectedWidth:  300,
			expectedHeight: 404,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			buf, _ := io.ReadAll(readFile(ImaginaryJpeg))
			img, err := Resize(buf, tc.opts)
			if err != nil {
				t.Errorf(CannotProcessImageS, err)
			}
			if img.Mime != ImageJPEG {
				t.Error(InvalidMimeType)
			}
			if err := assertSize(img.Body, tc.expectedWidth, tc.expectedHeight); err != nil {
				t.Errorf(InvalidImageSize, tc.expectedWidth, tc.expectedHeight)
			}
		})
	}
}

func TestImageFit(t *testing.T) {
	opts := ImageOptions{Width: 300, Height: 300}
	buf, _ := io.ReadAll(readFile(ImaginaryJpeg))

	img, err := Fit(buf, opts)
	if err != nil {
		t.Errorf(CannotProcessImageS, err)
	}
	if img.Mime != ImageJPEG {
		t.Error(InvalidMimeType)
	}
	// 550x740 -> 222.9x300
	if assertSize(img.Body, 223, 300) != nil {
		t.Errorf(InvalidImageSize, opts.Width, opts.Height)
	}
}

func TestImageAutoRotate(t *testing.T) {
	buf, _ := io.ReadAll(readFile(ImaginaryJpeg))
	img, err := AutoRotate(buf, ImageOptions{})
	if err != nil {
		t.Errorf(CannotProcessImageS, err)
	}
	if img.Mime != ImageJPEG {
		t.Error(InvalidMimeType)
	}
	if assertSize(img.Body, 550, 740) != nil {
		t.Errorf(InvalidImageSize, 550, 740)
	}
}

func TestImagePipelineOperations(t *testing.T) {
	width, height := 300, 260

	operations := PipelineOperations{
		PipelineOperation{
			Name: "crop",
			Params: map[string]interface{}{
				"width":  width,
				"height": height,
			},
		},
		PipelineOperation{
			Name: "convert",
			Params: map[string]interface{}{
				"type": "webp",
			},
		},
	}

	opts := ImageOptions{Operations: operations}
	buf, _ := io.ReadAll(readFile(ImaginaryJpeg))

	img, err := Pipeline(buf, opts)
	if err != nil {
		t.Errorf(CannotProcessImageS, err)
	}
	if img.Mime != "image/webp" {
		t.Error(InvalidMimeType)
	}
	if assertSize(img.Body, width, height) != nil {
		t.Errorf(InvalidImageSize, width, height)
	}
}

func TestCalculateDestinationFitDimension(t *testing.T) {
	cases := []struct {
		// Image
		imageWidth  int
		imageHeight int

		// User parameter
		optionWidth  int
		optionHeight int

		// Expect
		fitWidth  int
		fitHeight int
	}{

		// Leading Width
		{1280, 1000, 710, 9999, 710, 555},
		{1279, 1000, 710, 9999, 710, 555},
		{900, 500, 312, 312, 312, 173}, // rounding down
		{900, 500, 313, 313, 313, 174}, // rounding up

		// Leading height
		{1299, 2000, 710, 999, 649, 999},
		{1500, 2000, 710, 999, 710, 947},
	}

	for _, tc := range cases {
		fitWidth, fitHeight := calculateDestinationFitDimension(tc.imageWidth, tc.imageHeight, tc.optionWidth, tc.optionHeight)
		if fitWidth != tc.fitWidth || fitHeight != tc.fitHeight {
			t.Errorf(
				"Fit dimensions calculation failure\nExpected : %d/%d (width/height)\nActual   : %d/%d (width/height)\n%+v",
				tc.fitWidth, tc.fitHeight, fitWidth, fitHeight, tc,
			)
		}
	}

}
