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
	"testing"

	"github.com/h2non/bimg"
)

func TestExtractImageTypeFromMime(t *testing.T) {
	files := []struct {
		mime     string
		expected string
	}{
		{"image/jpeg", "jpeg"},
		{"/png", "png"},
		{"png", ""},
		{"multipart/form-data; encoding=utf-8", "form-data"},
		{"", ""},
	}

	for _, file := range files {
		if ExtractImageTypeFromMime(file.mime) != file.expected {
			t.Fatalf("Invalid mime type: %s != %s", file.mime, file.expected)
		}
	}
}

func TestIsImageTypeSupported(t *testing.T) {
	files := []struct {
		name     string
		expected bool
	}{
		{"image/jpeg", true},
		{"image/avif", true},
		{"image/png", true},
		{"image/webp", true},
		{"IMAGE/JPEG", true},
		{"png", false},
		{"multipart/form-data; encoding=utf-8", false},
		{"application/json", false},
		{"image/avif", bimg.IsImageTypeSupportedByVips(bimg.AVIF).Load},
		{"image/gif", bimg.IsImageTypeSupportedByVips(bimg.GIF).Load},
		{"image/svg+xml", bimg.IsImageTypeSupportedByVips(bimg.SVG).Load},
		{"image/svg", bimg.IsImageTypeSupportedByVips(bimg.SVG).Load},
		{"image/tiff", bimg.IsImageTypeSupportedByVips(bimg.TIFF).Load},
		{"application/pdf", bimg.IsImageTypeSupportedByVips(bimg.PDF).Load},
		{"text/plain", false},
		{"blablabla", false},
		{"", false},
	}

	for _, file := range files {
		if IsImageMimeTypeSupported(file.name) != file.expected {
			t.Fatalf("Invalid type: %s != %t", file.name, file.expected)
		}
	}
}

func TestImageType(t *testing.T) {
	files := []struct {
		name     string
		expected bimg.ImageType
	}{
		{"avif", bimg.AVIF},
		{"jpeg", bimg.JPEG},
		{"png", bimg.PNG},
		{"webp", bimg.WEBP},
		{"tiff", bimg.TIFF},
		{"gif", bimg.GIF},
		{"svg", bimg.SVG},
		{"pdf", bimg.PDF},
		{"multipart/form-data; encoding=utf-8", bimg.UNKNOWN},
		{"json", bimg.UNKNOWN},
		{"text", bimg.UNKNOWN},
		{"blablabla", bimg.UNKNOWN},
		{"", bimg.UNKNOWN},
	}

	for _, file := range files {
		if ImageType(file.name) != file.expected {
			t.Fatalf("Invalid type: %s != %s", file.name, bimg.ImageTypes[file.expected])
		}
	}
}

func TestGetImageMimeType(t *testing.T) {
	files := []struct {
		name     bimg.ImageType
		expected string
	}{
		{bimg.AVIF, "image/avif"},
		{bimg.JPEG, "image/jpeg"},
		{bimg.PNG, "image/png"},
		{bimg.WEBP, "image/webp"},
		{bimg.TIFF, "image/tiff"},
		{bimg.GIF, "image/gif"},
		{bimg.PDF, "application/pdf"},
		{bimg.SVG, "image/svg+xml"},
		{bimg.UNKNOWN, "image/jpeg"},
	}

	for _, file := range files {
		if GetImageMimeType(file.name) != file.expected {
			t.Fatalf("Invalid type: %s != %s", bimg.ImageTypes[file.name], file.expected)
		}
	}
}
