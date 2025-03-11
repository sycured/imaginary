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

const ImaginaryJpeg = "imaginary.jpg"
const InvalidMimeType = "Invalid image MIME type"

func TestImageResize(t *testing.T) {
	t.Run("Width and Height defined", func(t *testing.T) {
		opts := ImageOptions{Width: 300, Height: 300}
		buf, _ := io.ReadAll(readFile(ImaginaryJpeg))

		img, err := Resize(buf, opts)
		if err != nil {
			t.Errorf("Cannot process image: %s", err)
		}
		if img.Mime != ImageJPEG {
			t.Error(InvalidMimeType)
		}
		if assertSize(img.Body, opts.Width, opts.Height) != nil {
			t.Errorf("Invalid image size, expected: %dx%d", opts.Width, opts.Height)
		}
	})

	t.Run("Width defined", func(t *testing.T) {
		opts := ImageOptions{Width: 300}
		buf, _ := io.ReadAll(readFile(ImaginaryJpeg))

		img, err := Resize(buf, opts)
		if err != nil {
			t.Errorf("Cannot process image: %s", err)
		}
		if img.Mime != ImageJPEG {
			t.Error(InvalidMimeType)
		}
		if err := assertSize(img.Body, 300, 404); err != nil {
			t.Error(err)
		}
	})

	t.Run("Width defined with NoCrop=false", func(t *testing.T) {
		opts := ImageOptions{Width: 300, NoCrop: false, IsDefinedField: IsDefinedField{NoCrop: true}}
		buf, _ := io.ReadAll(readFile(ImaginaryJpeg))

		img, err := Resize(buf, opts)
		if err != nil {
			t.Errorf("Cannot process image: %s", err)
		}
		if img.Mime != ImageJPEG {
			t.Error(InvalidMimeType)
		}

		// The original image is 550x740
		if err := assertSize(img.Body, 300, 740); err != nil {
			t.Error(err)
		}
	})

	t.Run("Width defined with NoCrop=true", func(t *testing.T) {
		opts := ImageOptions{Width: 300, NoCrop: true, IsDefinedField: IsDefinedField{NoCrop: true}}
		buf, _ := io.ReadAll(readFile(ImaginaryJpeg))

		img, err := Resize(buf, opts)
		if err != nil {
			t.Errorf("Cannot process image: %s", err)
		}
		if img.Mime != ImageJPEG {
			t.Error(InvalidMimeType)
		}

		// The original image is 550x740
		if err := assertSize(img.Body, 300, 404); err != nil {
			t.Error(err)
		}
	})

}

func TestImageFit(t *testing.T) {
	opts := ImageOptions{Width: 300, Height: 300}
	buf, _ := io.ReadAll(readFile(ImaginaryJpeg))

	img, err := Fit(buf, opts)
	if err != nil {
		t.Errorf("Cannot process image: %s", err)
	}
	if img.Mime != ImageJPEG {
		t.Error(InvalidMimeType)
	}
	// 550x740 -> 222.9x300
	if assertSize(img.Body, 223, 300) != nil {
		t.Errorf("Invalid image size, expected: %dx%d", opts.Width, opts.Height)
	}
}

func TestImageAutoRotate(t *testing.T) {
	buf, _ := io.ReadAll(readFile(ImaginaryJpeg))
	img, err := AutoRotate(buf, ImageOptions{})
	if err != nil {
		t.Errorf("Cannot process image: %s", err)
	}
	if img.Mime != ImageJPEG {
		t.Error(InvalidMimeType)
	}
	if assertSize(img.Body, 550, 740) != nil {
		t.Errorf("Invalid image size, expected: %dx%d", 550, 740)
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
		t.Errorf("Cannot process image: %s", err)
	}
	if img.Mime != "image/webp" {
		t.Error(InvalidMimeType)
	}
	if assertSize(img.Body, width, height) != nil {
		t.Errorf("Invalid image size, expected: %dx%d", width, height)
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
