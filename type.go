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
	"strings"

	"github.com/h2non/bimg"
)

const SVG = "svg"

// ExtractImageTypeFromMime returns the MIME image type.
func ExtractImageTypeFromMime(mime string) string {
	mime = strings.Split(mime, ";")[0]
	parts := strings.Split(mime, "/")
	if len(parts) < 2 {
		return ""
	}
	name := strings.Split(parts[1], "+")[0]
	return strings.ToLower(name)
}

// IsImageMimeTypeSupported returns true if the image MIME
// type is supported by bimg.
func IsImageMimeTypeSupported(mime string) bool {
	format := ExtractImageTypeFromMime(mime)

	// Some payloads may expose the MIME type for SVG as text/xml
	if format == "xml" {
		format = SVG
	}

	return bimg.IsTypeNameSupported(format)
}

// ImageType returns the image type based on the given image type alias.
func ImageType(name string) bimg.ImageType {
	switch strings.ToLower(name) {
	case "avif":
		return bimg.AVIF
	case "gif":
		return bimg.GIF
	case "jpeg":
		return bimg.JPEG
	case "pdf":
		return bimg.PDF
	case "png":
		return bimg.PNG
	case SVG:
		return bimg.SVG
	case "tiff":
		return bimg.TIFF
	case "webp":
		return bimg.WEBP
	default:
		return bimg.UNKNOWN
	}
}

// GetImageMimeType returns the MIME type based on the given image type code.
func GetImageMimeType(code bimg.ImageType) string {
	switch code {
	case bimg.AVIF:
		return "image/avif"
	case bimg.GIF:
		return "image/gif"
	case bimg.PDF:
		return "application/pdf"
	case bimg.PNG:
		return "image/png"
	case bimg.SVG:
		return "image/svg+xml"
	case bimg.TIFF:
		return "image/tiff"
	case bimg.WEBP:
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
