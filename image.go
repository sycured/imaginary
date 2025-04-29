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

// @Summary Get image info
// @Description Returns metadata information about the image
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file to analyze"
// @Success 200 {object} ImageInfo
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /info [post]
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

// @Summary Resize image
// @Description Resizes an image to the specified dimensions
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param width query int false "Width of the output image"
// @Param height query int false "Height of the output image"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Param quality query int false "Quality of the output image (1-100)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /resize [post]
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

// @Summary Fit image
// @Description Fits an image within the specified dimensions while maintaining aspect ratio
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param width query int true "Width constraint"
// @Param height query int true "Height constraint"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Param quality query int false "Quality of the output image (1-100)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /fit [post]
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

// @Summary Enlarge image
// @Description Enlarges an image to the specified dimensions
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param width query int true "Width of the output image"
// @Param height query int true "Height of the output image"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Param quality query int false "Quality of the output image (1-100)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /enlarge [post]
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

// @Summary Extract area from image
// @Description Extracts a portion of the image with the specified dimensions
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param top query int false "Top offset for extraction"
// @Param left query int false "Left offset for extraction"
// @Param areawidth query int true "Width of the area to extract"
// @Param areaheight query int true "Height of the area to extract"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /extract [post]
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

// @Summary Crop image
// @Description Crops an image to the specified dimensions
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param width query int false "Width of the output image"
// @Param height query int false "Height of the output image"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Param quality query int false "Quality of the output image (1-100)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /crop [post]
func Crop(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewError(MissingHeightWidth, http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	opts.Crop = true
	return Process(buf, opts)
}

// @Summary Smart crop image
// @Description Intelligently crops an image to the specified dimensions
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param width query int false "Width of the output image"
// @Param height query int false "Height of the output image"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Param quality query int false "Quality of the output image (1-100)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /smartcrop [post]
func SmartCrop(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewError(MissingHeightWidth, http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	opts.Crop = true
	opts.Gravity = bimg.GravitySmart
	return Process(buf, opts)
}

// @Summary Rotate image
// @Description Rotates an image by the specified angle
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param rotate query int true "Rotation angle (90, 180, 270, 360)"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /rotate [post]
func Rotate(buf []byte, o ImageOptions) (Image, error) {
	if o.Rotate == 0 {
		return Image{}, NewError("Missing required param: rotate", http.StatusBadRequest)
	}

	opts := BimgOptions(o)
	return Process(buf, opts)
}

// @Summary Auto-rotate image
// @Description Automatically rotates an image based on EXIF orientation
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /autorotate [post]
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

// @Summary Flip image vertically
// @Description Flips an image vertically
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /flip [post]
func Flip(buf []byte, o ImageOptions) (Image, error) {
	opts := BimgOptions(o)
	opts.Flip = true
	return Process(buf, opts)
}

// @Summary Flip image horizontally
// @Description Flips an image horizontally
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /flop [post]
func Flop(buf []byte, o ImageOptions) (Image, error) {
	opts := BimgOptions(o)
	opts.Flop = true
	return Process(buf, opts)
}

// @Summary Create thumbnail
// @Description Creates a thumbnail of the image
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param width query int false "Width of the thumbnail"
// @Param height query int false "Height of the thumbnail"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /thumbnail [post]
func Thumbnail(buf []byte, o ImageOptions) (Image, error) {
	if o.Width == 0 && o.Height == 0 {
		return Image{}, NewError("Missing required params: width or height", http.StatusBadRequest)
	}

	return Process(buf, BimgOptions(o))
}

// @Summary Zoom image
// @Description Zooms an image by the specified factor
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param factor query number true "Zoom factor"
// @Param top query int false "Top offset for zoom area"
// @Param left query int false "Left offset for zoom area"
// @Param areawidth query int false "Width of the area to zoom"
// @Param areaheight query int false "Height of the area to zoom"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /zoom [post]
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

// @Summary Convert image format
// @Description Converts an image to a different format
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param type query string true "Output image format (jpeg, png, webp, etc.)"
// @Param quality query int false "Quality of the output image (1-100)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /convert [post]
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

// @Summary Add text watermark
// @Description Adds a text watermark to an image
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param text query string true "Watermark text"
// @Param font query string false "Font name and size (e.g., 'sans 12')"
// @Param opacity query number false "Opacity of the watermark (0.0-1.0)"
// @Param color query string false "Color of the watermark (R,G,B)"
// @Param textwidth query int false "Width of the text area"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /watermark [post]
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

// @Summary Add image watermark
// @Description Adds an image watermark to another image
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param image query string true "URL of the watermark image"
// @Param left query int false "Left offset for watermark"
// @Param top query int false "Top offset for watermark"
// @Param opacity query number false "Opacity of the watermark (0.0-1.0)"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /watermarkimage [post]
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

// @Summary Apply Gaussian blur
// @Description Applies Gaussian blur to an image
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param sigma query number true "Sigma parameter for Gaussian blur"
// @Param minampl query number false "Minimum amplitude"
// @Param type query string false "Output image format (jpeg, png, webp, etc.)"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /blur [post]
func GaussianBlur(buf []byte, o ImageOptions) (Image, error) {
	if o.Sigma == 0 && o.MinAmpl == 0 {
		return Image{}, NewError("Missing required param: sigma or minampl", http.StatusBadRequest)
	}
	opts := BimgOptions(o)
	return Process(buf, opts)
}

// @Summary Apply multiple operations
// @Description Applies a pipeline of operations to an image
// @Accept multipart/form-data
// @Produce image/*
// @Param file formData file true "Image file to process"
// @Param operations query string true "JSON array of operations to apply"
// @Success 200 {file} binary "Processed image"
// @Failure 400 {object} Error "Bad request"
// @Failure 404 {object} Error "Not found"
// @Failure 401 {object} Error "Unauthorized"
// @Failure 406 {object} Error "Not acceptable"
// @Failure 422 {object} Error "Unprocessable entity"
// @Router /pipeline [post]
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
