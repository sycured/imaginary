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
	"net/http"
	"net/url"
)

type ImageSourceType string
type ImageSourceFactoryFunction func(*SourceConfig) ImageSource

type SourceConfig struct {
	AuthForwarding     bool
	Authorization      string
	MountPath          string
	Type               ImageSourceType
	ForwardHeaders     []string
	SrcResponseHeaders []string
	AllowedOrigins     []*url.URL
	MaxAllowedSize     int
	AllowInsecureSSL   bool
}

var imageSourceMap = make(map[ImageSourceType]ImageSource)
var imageSourceFactoryMap = make(map[ImageSourceType]ImageSourceFactoryFunction)

type ImageSource interface {
	Matches(*http.Request) bool
	GetImage(*http.Request) ([]byte, http.Header, error)
}

func RegisterSource(sourceType ImageSourceType, factory ImageSourceFactoryFunction) {
	imageSourceFactoryMap[sourceType] = factory
}

func LoadSources(o ServerOptions) {
	for name, factory := range imageSourceFactoryMap {
		imageSourceMap[name] = factory(&SourceConfig{
			Type:               name,
			MountPath:          o.Mount,
			AuthForwarding:     o.AuthForwarding,
			Authorization:      o.Authorization,
			AllowedOrigins:     o.AllowedOrigins,
			MaxAllowedSize:     o.MaxAllowedSize,
			ForwardHeaders:     o.ForwardHeaders,
			SrcResponseHeaders: o.SrcResponseHeaders,
			AllowInsecureSSL:   o.AllowInsecureSSL,
		})
	}
}

func MatchSource(req *http.Request) ImageSource {
	for _, source := range imageSourceMap {
		if source.Matches(req) {
			return source
		}
	}
	return nil
}
