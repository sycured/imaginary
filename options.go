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
	"strconv"
	"strings"

	"github.com/h2non/bimg"
)

// ImageOptions represent all the supported image transformation params as first level members
type ImageOptions struct {
	IsDefinedField

	Width         int
	Height        int
	AreaWidth     int
	AreaHeight    int
	Quality       int
	Compression   int
	Rotate        int
	Top           int
	Left          int
	Margin        int
	Factor        int
	DPI           int
	TextWidth     int
	Flip          bool
	Flop          bool
	Force         bool
	Embed         bool
	NoCrop        bool
	NoReplicate   bool
	NoRotation    bool
	NoProfile     bool
	StripMetadata bool
	Opacity       float32
	Sigma         float64
	MinAmpl       float64
	Text          string
	Image         string
	Font          string
	Type          string
	AspectRatio   string
	Color         []uint8
	Background    []uint8
	Interlace     bool
	Speed         int
	Extend        bimg.Extend
	Gravity       bimg.Gravity
	Colorspace    bimg.Interpretation
	Operations    PipelineOperations
}

// IsDefinedField holds boolean ImageOptions fields. If true it means the field was specified in the request. This
// metadata allows for sane usage of default (false) values.
type IsDefinedField struct {
	Flip          bool
	Flop          bool
	Force         bool
	Embed         bool
	NoCrop        bool
	NoReplicate   bool
	NoRotation    bool
	NoProfile     bool
	StripMetadata bool
	Interlace     bool
	Palette       bool
}

// PipelineOperation represents the structure for an operation field.
type PipelineOperation struct {
	Name          string                 `json:"operation"`
	IgnoreFailure bool                   `json:"ignore_failure"`
	Params        map[string]interface{} `json:"params"`
	ImageOptions  ImageOptions           `json:"-"`
	Operation     Operation              `json:"-"`
}

// PipelineOperations defines the expected interface for a list of operations.
type PipelineOperations []PipelineOperation

func transformByAspectRatio(params map[string]interface{}) (width, height int) {
	width, _ = coerceTypeInt(params["width"])
	height, _ = coerceTypeInt(params["height"])

	aspectRatio, ok := params["aspectratio"].(map[string]int)
	if !ok {
		return
	}

	if width != 0 {
		height = calculateFromAspectRatio(width, aspectRatio["height"], aspectRatio["width"])
	} else {
		width = calculateFromAspectRatio(height, aspectRatio["width"], aspectRatio["height"])
	}

	return
}

func calculateFromAspectRatio(x int, ratioA int, ratioB int) int {
	res := float32(x) * (float32(ratioA) / float32(ratioB))
	return int(res)
}

func parseAspectRatio(val string) map[string]int {
	val = strings.TrimSpace(strings.ToLower(val))
	slicedVal := strings.Split(val, ":")

	if len(slicedVal) < 2 {
		return nil
	}

	width, _ := strconv.Atoi(slicedVal[0])
	height, _ := strconv.Atoi(slicedVal[1])

	return map[string]int{
		"width":  width,
		"height": height,
	}
}

func shouldTransformByAspectRatio(height, width int) bool {

	// override aspect ratio parameters if width and height is given or not given at all
	if (width != 0 && height != 0) || (width == 0 && height == 0) {
		return false
	}

	return true
}

// BimgOptions creates a new bimg compatible options struct mapping the fields properly
func BimgOptions(o ImageOptions) bimg.Options {
	opts := bimg.Options{
		Width:          o.Width,
		Height:         o.Height,
		Flip:           o.Flip,
		Flop:           o.Flop,
		Quality:        o.Quality,
		Compression:    o.Compression,
		NoAutoRotate:   o.NoRotation,
		NoProfile:      o.NoProfile,
		Force:          o.Force,
		Gravity:        o.Gravity,
		Embed:          o.Embed,
		Extend:         o.Extend,
		Interpretation: o.Colorspace,
		StripMetadata:  o.StripMetadata,
		Type:           ImageType(o.Type),
		Rotate:         bimg.Angle(o.Rotate),
		Interlace:      o.Interlace,
		Palette:        o.Palette,
		Speed:          o.Speed,
	}

	if len(o.Background) != 0 {
		opts.Background = bimg.Color{R: o.Background[0], G: o.Background[1], B: o.Background[2]}
	}

	if shouldTransformByAspectRatio(opts.Height, opts.Width) && o.AspectRatio != "" {
		params := make(map[string]interface{})
		params["height"] = opts.Height
		params["width"] = opts.Width
		params["aspectratio"] = parseAspectRatio(o.AspectRatio)

		opts.Width, opts.Height = transformByAspectRatio(params)
	}

	if o.Sigma > 0 || o.MinAmpl > 0 {
		opts.GaussianBlur = bimg.GaussianBlur{
			Sigma:   o.Sigma,
			MinAmpl: o.MinAmpl,
		}
	}

	return opts
}
