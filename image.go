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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/h2non/bimg"
)

const MissingHeightWidth = "Missing required param: height or width"

// OperationsMap defines the allowed image transformation operations listed by name.
// Used for pipeline image processing.
var OperationsMap = map[string]Operation{
	"crop":           Crop,
	"resize":         Resize,
	"enlarge":        Enlarge,
	"extract":        Extract,
	"rotate":         Rotate,
	"autorotate":     AutoRotate,
	"flip":           Flip,
	"flop":           Flop,
	"thumbnail":      Thumbnail,
	"zoom":           Zoom,
	"convert":        Convert,
	"watermark":      Watermark,
	"watermarkImage": WatermarkImage,
	"blur":           GaussianBlur,
	"smartcrop":      SmartCrop,
	"fit":            Fit,
}

// Image stores an image binary buffer and its MIME type
type Image struct {
	Body []byte
	Mime string
}

// Operation implements an image transformation runnable interface
type Operation func([]byte, ImageOptions) (Image, error)

// Run performs the image transformation
func (o Operation) Run(buf []byte, opts ImageOptions) (Image, error) {
	return o(buf, opts)
}

// ImageInfo represents an image details and additional metadata
type ImageInfo struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Type        string `json:"type"`
	Space       string `json:"space"`
	Alpha       bool   `json:"hasAlpha"`
	Profile     bool   `json:"hasProfile"`
	Channels    int    `json:"channels"`
	Orientation int    `json:"orientation"`
}

func Info(buf []byte, _ ImageOptions) (Image, error) {
	// We're not handling an image here, but we reused the struct.
	// An interface will be definitively better here.
	image := Image{Mime: "application/json"}

	meta, err := bimg.Metadata(buf)
	if err != nil {
		return image, NewError("Cannot retrieve image metadata: %s"+err.Error(), http.StatusBadRequest)
	}

	info := ImageInfo{
		Width:       meta.Size.Width,
		Height:      meta.Size.Height,
		Type:        meta.Type,
		Space:       meta.Space,
		Alpha:       meta.Alpha,
		Profile:     meta.Profile,
		Channels:    meta.Channels,
		Orientation: meta.Orientation,
	}

	body, _ := json.Marshal(info)
	image.Body = body

	return image, nil
}

func Resize(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewError(MissingHeightWidth, http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	opts.Embed = true

	if o.IsDefinedField.NoCrop {
		opts.Crop = !o.NoCrop
	}

	return Process(buf, opts)
}

func Fit(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 || o.Height == 0 {
		return Image{}, NewError("Missing required params: height, width", http.StatusBadRequest)
	}

	metadata, err := bimg.Metadata(buf)
	if err != nil {
		return Image{}, err
	}

	dims := metadata.Size

	if dims.Width == 0 || dims.Height == 0 {
		return Image{}, NewError("Width or height of requested image is zero", http.StatusNotAcceptable)
	}

	// metadata.Orientation
	// 0: no EXIF orientation
	// 1: CW 0
	// 2: CW 0, flip horizontal
	// 3: CW 180
	// 4: CW 180, flip horizontal
	// 5: CW 90, flip horizontal
	// 6: CW 270
	// 7: CW 270, flip horizontal
	// 8: CW 90

	var originHeight, originWidth int
	var fitHeight, fitWidth *int
	if o.NoRotation || (metadata.Orientation <= 4) {
		originHeight = dims.Height
		originWidth = dims.Width
		fitHeight = &o.Height
		fitWidth = &o.Width
	} else {
		// width/height will be switched with auto rotation
		originWidth = dims.Height
		originHeight = dims.Width
		fitWidth = &o.Height
		fitHeight = &o.Width
	}

	*fitWidth, *fitHeight = calculateDestinationFitDimension(originWidth, originHeight, *fitWidth, *fitHeight)

	opts := BimgOptions(o)
	opts.Embed = true

	return Process(buf, opts)
}

// calculateDestinationFitDimension calculates the fit area based on the image and desired fit dimensions
func calculateDestinationFitDimension(imageWidth, imageHeight, fitWidth, fitHeight int) (int, int) {
	if imageWidth*fitHeight > fitWidth*imageHeight {
		// constrained by width
		fitHeight = int(math.Round(float64(fitWidth) * float64(imageHeight) / float64(imageWidth)))
	} else {
		// constrained by height
		fitWidth = int(math.Round(float64(fitHeight) * float64(imageWidth) / float64(imageHeight)))
	}

	return fitWidth, fitHeight
}

func Enlarge(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 || o.Height == 0 {
		return Image{}, NewError("Missing required params: height, width", http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	opts.Enlarge = true

	// Since both width & height is required, we allow cropping by default.
	opts.Crop = !o.NoCrop

	return Process(buf, opts)
}

func Extract(buf []byte, o ImageOptions) (Image, error) {
	if o.AreaWidth == 0 || o.AreaHeight == 0 {
		return Image{}, NewError("Missing required params: areawidth or areaheight", http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	opts.Top = o.Top
	opts.Left = o.Left
	opts.AreaWidth = o.AreaWidth
	opts.AreaHeight = o.AreaHeight

	return Process(buf, opts)
}

func Crop(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewError(MissingHeightWidth, http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	opts.Crop = true
	return Process(buf, opts)
}

func SmartCrop(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewError(MissingHeightWidth, http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	opts.Crop = true
	opts.Gravity = bimg.GravitySmart
	return Process(buf, opts)
}

func Rotate(buf []byte, o ImageOptions) (Image, error) {
	if o.Rotate == 0 {
		return Image{}, NewError("Missing required param: rotate", http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	return Process(buf, opts)
}

func AutoRotate(buf []byte, _ ImageOptions) (out Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch value := r.(type) {
			case error:
				err = value
			case string:
				err = errors.New(value)
			default:
				err = errors.New("libvips internal error")
			}
			out = Image{}
		}
	}()

	// Resize image via bimg
	ibuf, err := bimg.NewImage(buf).AutoRotate()
	if err != nil {
		return Image{}, err
	}

	mime := GetImageMimeType(bimg.DetermineImageType(ibuf))
	return Image{Body: ibuf, Mime: mime}, nil
}

func Flip(buf []byte, o ImageOptions) (Image, error) {
	opts := BimgOptions(o)
	opts.Flip = true
	return Process(buf, opts)
}

func Flop(buf []byte, o ImageOptions) (Image, error) {
	opts := BimgOptions(o)
	opts.Flop = true
	return Process(buf, opts)
}

func Thumbnail(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewError("Missing required params: width or height", http.StatusBadRequest)
	}

	return Process(buf, BimgOptions(o))
}

func Zoom(buf []byte, o ImageOptions) (Image, error) {
	if o.Factor == 0 {
		return Image{}, NewError("Missing required param: factor", http.StatusBadRequest)
	}

	opts := BimgOptions(o)

	if o.Top > 0 || o.Left > 0 {
		if o.AreaWidth == 0 && o.AreaHeight == 0 {
			return Image{}, NewError("Missing required params: areawidth, areaheight", http.StatusBadRequest)
		}

		opts.Top = o.Top
		opts.Left = o.Left
		opts.AreaWidth = o.AreaWidth
		opts.AreaHeight = o.AreaHeight

		if o.IsDefinedField.NoCrop {
			opts.Crop = !o.NoCrop
		}
	}

	opts.Zoom = o.Factor
	return Process(buf, opts)
}

func Convert(buf []byte, o ImageOptions) (Image, error) {
	if o.Type == "" {
		return Image{}, NewError("Missing required param: type", http.StatusBadRequest)
	}
	if ImageType(o.Type) == bimg.UNKNOWN {
		return Image{}, NewError("Invalid image type: "+o.Type, http.StatusBadRequest)
	}
	opts := BimgOptions(o)

	return Process(buf, opts)
}

func Watermark(buf []byte, o ImageOptions) (Image, error) {
	if o.Text == "" {
		return Image{}, NewError("Missing required param: text", http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	opts.Watermark.DPI = o.DPI
	opts.Watermark.Text = o.Text
	opts.Watermark.Font = o.Font
	opts.Watermark.Margin = o.Margin
	opts.Watermark.Width = o.TextWidth
	opts.Watermark.Opacity = o.Opacity
	opts.Watermark.NoReplicate = o.NoReplicate

	if len(o.Color) > 2 {
		opts.Watermark.Background = bimg.Color{R: o.Color[0], G: o.Color[1], B: o.Color[2]}
	}

	return Process(buf, opts)
}

func WatermarkImage(buf []byte, o ImageOptions) (Image, error) {
	if o.Image == "" {
		return Image{}, NewError("Missing required param: image", http.StatusBadRequest)
	}
	response, err := http.Get(o.Image)
	if err != nil {
		return Image{}, NewError(fmt.Sprintf("Unable to retrieve watermark image. %s", o.Image), http.StatusBadRequest)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	bodyReader := io.LimitReader(response.Body, 1e6)

	imageBuf, err := io.ReadAll(bodyReader)
	if len(imageBuf) == 0 {
		errMessage := "Unable to read watermark image"

		if err != nil {
			errMessage = fmt.Sprintf("%s. %s", errMessage, err.Error())
		}

		return Image{}, NewError(errMessage, http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	opts.WatermarkImage.Left = o.Left
	opts.WatermarkImage.Top = o.Top
	opts.WatermarkImage.Buf = imageBuf
	opts.WatermarkImage.Opacity = o.Opacity

	return Process(buf, opts)
}

func GaussianBlur(buf []byte, o ImageOptions) (Image, error) {
	if o.Sigma == 0 && o.MinAmpl == 0 {
		return Image{}, NewError("Missing required param: sigma or minampl", http.StatusBadRequest)
	}
	opts := BimgOptions(o)
	return Process(buf, opts)
}

func Pipeline(buf []byte, o ImageOptions) (Image, error) {
	if len(o.Operations) == 0 {
		return Image{}, NewError("Missing or invalid pipeline operations JSON", http.StatusBadRequest)
	}
	if len(o.Operations) > 10 {
		return Image{}, NewError("Maximum allowed pipeline operations exceeded", http.StatusBadRequest)
	}

	// Validate and built operations
	for i, operation := range o.Operations {
		// Validate supported operation name
		var exists bool
		if operation.Operation, exists = OperationsMap[operation.Name]; !exists {
			return Image{}, NewError(fmt.Sprintf("Unsupported operation name: %s", operation.Name), http.StatusBadRequest)
		}

		// Parse and construct operation options
		var err error
		operation.ImageOptions, err = buildParamsFromOperation(operation)
		if err != nil {
			return Image{}, err
		}

		// Mutate list by value
		o.Operations[i] = operation
	}

	var image Image
	var err error

	// Reduce image by running multiple operations
	image = Image{Body: buf}
	for _, operation := range o.Operations {
		var curImage Image
		curImage, err = operation.Operation(image.Body, operation.ImageOptions)
		if err != nil && !operation.IgnoreFailure {
			return Image{}, err
		}
		if operation.IgnoreFailure {
			err = nil
		}
		if err == nil {
			image = curImage
		}
	}

	return image, err
}

func Process(buf []byte, opts bimg.Options) (out Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch value := r.(type) {
			case error:
				err = value
			case string:
				err = errors.New(value)
			default:
				err = errors.New("libvips internal error")
			}
			out = Image{}
		}
	}()

	// Resize image via bimg
	ibuf, err := bimg.Resize(buf, opts)

	// Handle specific type encode errors gracefully
	if err != nil && strings.Contains(err.Error(), "encode") && (opts.Type == bimg.WEBP || opts.Type == bimg.HEIF) {
		// Always fallback to JPEG
		opts.Type = bimg.JPEG
		ibuf, err = bimg.Resize(buf, opts)
	}

	if err != nil {
		return Image{}, err
	}

	mime := GetImageMimeType(bimg.DetermineImageType(ibuf))
	return Image{Body: ibuf, Mime: mime}, nil
}
