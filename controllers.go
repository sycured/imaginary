package main

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/h2non/bimg"
	"github.com/h2non/filetype"
)

const (
	ContentType     = "Content-Type"
	ContentTypeJSON = "application/json"
	ImageJPEG       = "image/jpeg"
	ImagePNG        = "image/png"
	ImageSVG        = "image/svg+xml"
	ImageWebP       = "image/webp"
	JPEG            = "jpeg"
	PNG             = "png"
	WebP            = "webp"
)

func indexController(o ServerOptions) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path.Join(o.PathPrefix, "/") {
			ErrorReply(r, w, ErrNotFound, ServerOptions{})
			return
		}

		body, _ := json.Marshal(Versions{
			Version,
			bimg.Version,
			bimg.VipsVersion,
		})
		w.Header().Set(ContentType, ContentTypeJSON)
		_, _ = w.Write(body)
	}
}

func healthController(w http.ResponseWriter, r *http.Request) {
	health := GetHealthStats()
	body, _ := json.Marshal(health)
	w.Header().Set(ContentType, ContentTypeJSON)
	_, _ = w.Write(body)
}

func imageController(o ServerOptions, operation Operation) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var imageSource = MatchSource(req)
		if imageSource == nil {
			ErrorReply(req, w, ErrMissingImageSource, o)
			return
		}

		buf, err := imageSource.GetImage(req)
		if err != nil {
			if xerr, ok := err.(Error); ok {
				ErrorReply(req, w, xerr, o)
			} else {
				ErrorReply(req, w, NewError(err.Error(), http.StatusBadRequest), o)
			}
			return
		}

		if len(buf) == 0 {
			ErrorReply(req, w, ErrEmptyBody, o)
			return
		}

		imageHandler(w, req, buf, operation, o)
	}
}

func determineAcceptMimeType(accept string) string {
	for _, v := range strings.Split(accept, ",") {
		mediaType, _, _ := mime.ParseMediaType(v)
		switch mediaType {
		case ImageWebP:
			return WebP
		case ImagePNG:
			return PNG
		case ImageJPEG:
			return JPEG
		}
	}

	return ""
}

func imageHandler(w http.ResponseWriter, r *http.Request, buf []byte, operation Operation, o ServerOptions) {
	mimeType, err := inferMimeType(buf)
	if err != nil || !IsImageMimeTypeSupported(mimeType) {
		ErrorReply(r, w, ErrUnsupportedMedia, o)
		return
	}

	opts, vary, err := processImageOptions(r)
	if err != nil {
		ErrorReply(r, w, NewError(err.Error(), http.StatusBadRequest), o)
		return
	}

	if err := validateImageSize(buf, o); err != nil {
		ErrorReply(r, w, NewError(err.Error(), http.StatusBadRequest), o)
		return
	}

	image, err := operation.Run(buf, opts)
	if err != nil {
		handleProcessingError(w, r, vary, err, o)
		return
	}

	sendResponse(w, image, vary, o)
}

//nolint:unparam
func inferMimeType(buf []byte) (string, error) {
	mimeType := http.DetectContentType(buf)
	if mimeType == "application/octet-stream" {
		kind, err := filetype.Get(buf)
		if err == nil && kind.MIME.Value != "" {
			mimeType = kind.MIME.Value
		}
	}
	if strings.Contains(mimeType, "text/plain") && len(buf) > 8 && bimg.IsSVGImage(buf) {
		mimeType = ImageSVG
	}
	return mimeType, nil
}

func processImageOptions(r *http.Request) (ImageOptions, string, error) {
	opts, err := buildParamsFromQuery(r.URL.Query())
	if err != nil {
		return ImageOptions{}, "", NewError("Error while processing parameters, "+err.Error(), http.StatusBadRequest)
	}

	vary := ""
	if opts.Type == "auto" {
		opts.Type = determineAcceptMimeType(r.Header.Get("Accept"))
		vary = "Accept"
	} else if opts.Type != "" && ImageType(opts.Type) == 0 {
		return ImageOptions{}, "", ErrOutputFormat
	}
	return opts, vary, nil
}

func validateImageSize(buf []byte, o ServerOptions) error {
	sizeInfo, err := bimg.Size(buf)
	if err != nil {
		return NewError("Error while processing the image: "+err.Error(), http.StatusBadRequest)
	}
	imgResolution := float64(sizeInfo.Width) * float64(sizeInfo.Height)
	if (imgResolution / 1000000) > o.MaxAllowedPixels {
		return ErrResolutionTooBig
	}
	return nil
}

func handleProcessingError(w http.ResponseWriter, r *http.Request, vary string, err error, o ServerOptions) {
	if vary != "" {
		w.Header().Set("Vary", vary)
	}
	ErrorReply(r, w, NewError("Error while processing the image: "+err.Error(), http.StatusBadRequest), o)
}

func sendResponse(w http.ResponseWriter, image Image, vary string, o ServerOptions) {
	w.Header().Set("Content-Length", strconv.Itoa(len(image.Body)))
	w.Header().Set(ContentType, image.Mime)
	if image.Mime != ContentTypeJSON && o.ReturnSize {
		meta, err := bimg.Metadata(image.Body)
		if err == nil {
			w.Header().Set("Image-Width", strconv.Itoa(meta.Size.Width))
			w.Header().Set("Image-Height", strconv.Itoa(meta.Size.Height))
		}
	}
	if vary != "" {
		w.Header().Set("Vary", vary)
	}
	_, _ = w.Write(image.Body)
}

func formController(o ServerOptions) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		operations := []struct {
			name   string
			method string
			args   string
		}{
			{"Resize", "resize", "width=300&height=200&type=jpeg"},
			{"Force resize", "resize", "width=300&height=200&force=true"},
			{"Crop", "crop", "width=300&quality=95"},
			{"SmartCrop", "crop", "width=300&height=260&quality=95&gravity=smart"},
			{"Extract", "extract", "top=100&left=100&areawidth=300&areaheight=150"},
			{"Enlarge", "enlarge", "width=1440&height=900&quality=95"},
			{"Rotate", "rotate", "rotate=180"},
			{"AutoRotate", "autorotate", "quality=90"},
			{"Flip", "flip", ""},
			{"Flop", "flop", ""},
			{"Thumbnail", "thumbnail", "width=100"},
			{"Zoom", "zoom", "factor=2&areawidth=300&top=80&left=80"},
			{"Color space (black&white)", "resize", "width=400&height=300&colorspace=bw"},
			{"Add watermark", "watermark", "textwidth=100&text=Hello&font=sans%2012&opacity=0.5&color=255,200,50"},
			{"Convert format", "convert", "type=png"},
			{"Image metadata", "info", ""},
			{"Gaussian blur", "blur", "sigma=15.0&minampl=0.2"},
			{"Pipeline (image reduction via multiple transformations)", "pipeline", "operations=%5B%7B%22operation%22:%20%22crop%22,%20%22params%22:%20%7B%22width%22:%20300,%20%22height%22:%20260%7D%7D,%20%7B%22operation%22:%20%22convert%22,%20%22params%22:%20%7B%22type%22:%20%22webp%22%7D%7D%5D"}, //nolint:lll
		}

		html := "<html><body>"

		for _, form := range operations {
			html += fmt.Sprintf(`
		<h1>%s</h1>
		<form method="POST" action="%s?%s" enctype="multipart/form-data">
		<input type="file" name="file" />
		<input type="submit" value="Upload" />
		</form>`, path.Join(o.PathPrefix, form.name), path.Join(o.PathPrefix, form.method), form.args)
		}

		html += "</body></html>"

		w.Header().Set(ContentType, "text/html")
		_, _ = w.Write([]byte(html))
	}
}
