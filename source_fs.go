package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

const ImageSourceTypeFileSystem ImageSourceType = "fs"

type FileSystemImageSource struct {
	Config *SourceConfig
}

func NewFileSystemImageSource(config *SourceConfig) ImageSource {
	return &FileSystemImageSource{config}
}

func (s *FileSystemImageSource) Matches(r *http.Request) bool {
	file, err := s.getFileParam(r)
	if err != nil {
		return false
	}
	return r.Method == http.MethodGet && file != ""
}

func (s *FileSystemImageSource) GetImage(r *http.Request) ([]byte, http.Header, error) {
	file, err := s.getFileParam(r)
	if err != nil {
		return nil, nil, err
	}

	if file == "" {
		return nil, nil, ErrMissingParamFile
	}

	file, err = s.buildPath(file)
	if err != nil {
		return nil, nil, err
	}

	return s.read(file)
}

func (s *FileSystemImageSource) buildPath(file string) (string, error) {
	file = path.Clean(path.Join(s.Config.MountPath, file))
	if !strings.HasPrefix(file, s.Config.MountPath) {
		return "", ErrInvalidFilePath
	}
	return file, nil
}

func (s *FileSystemImageSource) read(file string) ([]byte, http.Header, error) {
	buf, err := os.ReadFile(file) //nolint:gosec
	if err != nil {
		return nil, nil, ErrInvalidFilePath
	}
	return buf, make(http.Header), nil
}

func (s *FileSystemImageSource) getFileParam(r *http.Request) (string, error) {
	unescaped, err := url.QueryUnescape(r.URL.Query().Get("file"))
	if err != nil {
		return "", fmt.Errorf("failed to unescape file param: %w", err)
	}

	return unescaped, nil
}

func init() {
	RegisterSource(ImageSourceTypeFileSystem, NewFileSystemImageSource)
}
