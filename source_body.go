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
	"mime/multipart"
	"net/http"
	"strings"
)

const formFieldName = "file"
const maxMemory int64 = 1024 * 1024 * 64

const ImageSourceTypeBody ImageSourceType = "payload"

type BodyImageSource struct {
	Config *SourceConfig
}

func NewBodyImageSource(config *SourceConfig) ImageSource {
	return &BodyImageSource{config}
}

func (s *BodyImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodPost || r.Method == http.MethodPut
}

func (s *BodyImageSource) GetImage(r *http.Request) ([]byte, http.Header, error) {
	var buf []byte
	var err error

	if isFormBody(r) {
		buf, err = readFormBody(r)
	} else {
		buf, err = readRawBody(r)
	}
	return buf, make(http.Header), err
}

func isFormBody(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/")
}

func readFormBody(r *http.Request) ([]byte, error) {
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return nil, err
	}

	file, _, err := r.FormFile(formFieldName)
	if err != nil {
		return nil, err
	}
	defer func(file multipart.File) {
		_ = file.Close()
	}(file)

	buf, err := io.ReadAll(file)
	if len(buf) == 0 {
		err = ErrEmptyBody
	}

	return buf, err
}

func readRawBody(r *http.Request) ([]byte, error) {
	return io.ReadAll(r.Body)
}

func init() {
	RegisterSource(ImageSourceTypeBody, NewBodyImageSource)
}
