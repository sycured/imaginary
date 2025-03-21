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
	"math"
	"net/url"
	"testing"

	"github.com/h2non/bimg"
)

const DidNotExpectError = "Did not expect error %s\n%+v"
const ExpectedError = "Expected an error to be thrown\nExpected: %s\nReceived: %s"
const ExpectedProperCoercion = "Expected proper coercion %s\n%+v\n%+v"
const epsilon = 0.0001

func TestReadParams(t *testing.T) {
	q := url.Values{}
	q.Set("width", "100")
	q.Add("height", "80")
	q.Add("noreplicate", "1")
	q.Add("opacity", "0.2")
	q.Add("text", "hello")
	q.Add("background", "255,10,20")
	q.Add("interlace", "true")

	params, err := buildParamsFromQuery(q)
	if err != nil {
		t.Errorf("Failed reading params, %s", err)
	}

	assert := params.Width == 100 &&
		params.Height == 80 &&
		params.NoReplicate == true &&
		params.Opacity == 0.2 &&
		params.Text == "hello" &&
		params.Background[0] == 255 &&
		params.Background[1] == 10 &&
		params.Background[2] == 20 &&
		params.Interlace == true

	if assert == false {
		t.Error("Invalid params")
	}
}

func TestParseParam(t *testing.T) {
	intCases := []struct {
		value    string
		expected int
	}{
		{"1", 1},
		{"0100", 100},
		{"-100", 100},
		{"99.02", 99},
		{"99.9", 100},
	}

	for _, test := range intCases {
		val, _ := parseInt(test.value)
		if val != test.expected {
			t.Errorf("Invalid param: %s != %d", test.value, test.expected)
		}
	}

	floatCases := []struct {
		value    string
		expected float64
	}{
		{"1.1", 1.1},
		{"01.1", 1.1},
		{"-1.10", 1.10},
		{"99.999999", 99.999999},
	}

	for _, test := range floatCases {
		val, _ := parseFloat(test.value)
		if val != test.expected {
			t.Errorf("Invalid param: %#v != %#v", val, test.expected)
		}
	}

	boolCases := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"1.1", false},
		{"-1", false},
		{"0", false},
		{"0.0", false},
		{"no", false},
		{"yes", false},
	}

	for _, test := range boolCases {
		val, _ := parseBool(test.value)
		if val != test.expected {
			t.Errorf("Invalid param: %#v != %#v", val, test.expected)
		}
	}
}

func TestParseColor(t *testing.T) {
	cases := []struct {
		value    string
		expected []uint8
	}{
		{"200,100,20", []uint8{200, 100, 20}},
		{"0,280,200", []uint8{0, 255, 200}},
		{" -1, 256 , 50", []uint8{0, 255, 50}},
		{" a, 20 , &hel0", []uint8{0, 20, 0}},
		{"", []uint8{}},
	}

	for _, color := range cases {
		c := parseColor(color.value)
		l := len(color.expected)

		if len(c) != l {
			t.Errorf("Invalid color length: %#v", c)
		}
		if l == 0 {
			continue
		}

		assert := c[0] == color.expected[0] &&
			c[1] == color.expected[1] &&
			c[2] == color.expected[2]

		if assert == false {
			t.Errorf("Invalid color schema: %#v <> %#v", color.expected, c)
		}
	}
}

func TestParseExtend(t *testing.T) {
	cases := []struct {
		value    string
		expected bimg.Extend
	}{
		{"white", bimg.ExtendWhite},
		{"black", bimg.ExtendBlack},
		{"copy", bimg.ExtendCopy},
		{"mirror", bimg.ExtendMirror},
		{"lastpixel", bimg.ExtendLast},
		{"background", bimg.ExtendBackground},
		{" BACKGROUND  ", bimg.ExtendBackground},
		{"invalid", bimg.ExtendMirror},
		{"", bimg.ExtendMirror},
	}

	for _, extend := range cases {
		c := parseExtendMode(extend.value)
		if c != extend.expected {
			t.Errorf("Invalid extend value : %d != %d", c, extend.expected)
		}
	}
}

func TestGravity(t *testing.T) {
	cases := []struct {
		gravityValue   string
		smartCropValue bool
	}{
		{gravityValue: "foo", smartCropValue: false},
		{gravityValue: "smart", smartCropValue: true},
	}

	for _, td := range cases {
		io, _ := buildParamsFromQuery(url.Values{"gravity": []string{td.gravityValue}})
		if (io.Gravity == bimg.GravitySmart) != td.smartCropValue {
			t.Errorf("Expected %t to be %t, test data: %+v", io.Gravity == bimg.GravitySmart, td.smartCropValue, td)
		}
	}
}

func TestReadMapParams(t *testing.T) {
	cases := []struct {
		params   map[string]interface{}
		expected ImageOptions
	}{
		{
			map[string]interface{}{
				"width":   100,
				"opacity": 0.1,
				"type":    "webp",
				"embed":   true,
				"gravity": "west",
				"color":   "255,200,150",
			},
			ImageOptions{
				Width:   100,
				Opacity: 0.1,
				Type:    "webp",
				Embed:   true,
				Gravity: bimg.GravityWest,
				Color:   []uint8{255, 200, 150},
			},
		},
	}

	for _, test := range cases {
		opts, err := buildParamsFromOperation(PipelineOperation{Params: test.params})
		if err != nil {
			t.Errorf("Error reading parameters %s", err)
			t.FailNow()
		}
		validateImageOptions(t, opts, test.expected)
	}
}

func validateImageOptions(t *testing.T, opts, expected ImageOptions) {
	if opts.Width != expected.Width {
		t.Errorf("Invalid width: %d != %d", opts.Width, expected.Width)
	}
	if opts.Opacity != expected.Opacity {
		t.Errorf("Invalid opacity: %v != %v", opts.Opacity, expected.Opacity)
	}
	if opts.Type != expected.Type {
		t.Errorf("Invalid type: %s != %s", opts.Type, expected.Type)
	}
	if opts.Embed != expected.Embed {
		t.Errorf("Invalid embed: %v != %v", opts.Embed, expected.Embed)
	}
	if opts.Gravity != expected.Gravity {
		t.Errorf("Invalid gravity: %v != %v", opts.Gravity, expected.Gravity)
	}
	if len(opts.Color) < 3 || len(expected.Color) < 3 ||
		opts.Color[0] != expected.Color[0] ||
		opts.Color[1] != expected.Color[1] ||
		opts.Color[2] != expected.Color[2] {
		t.Errorf("Invalid color: %v != %v", opts.Color, expected.Color)
	}
}

func TestParseFunctions(t *testing.T) {
	t.Run("parseBool", func(t *testing.T) {
		if r, err := parseBool("true"); r != true {
			t.Errorf("Expected string true to result a native type true %s", err)
		}

		if r, err := parseBool("false"); r != false {
			t.Errorf("Expected string false to result a native type false %s", err)
		}

		// A special case that we support
		if _, err := parseBool(""); err != nil {
			t.Errorf("Expected blank values to default to false, it didn't! %s", err)
		}

		if r, err := parseBool("foo"); err == nil {
			t.Errorf("Expected malformed values to result in an error, it didn't! %+v", r)
		}
	})
}

func TestBuildParamsFromOperation(t *testing.T) {
	op := PipelineOperation{
		Params: map[string]interface{}{
			"width":      200,
			"opacity":    2.2,
			"force":      true,
			"stripmeta":  false,
			"type":       "jpeg",
			"background": "255,12,3",
		},
	}

	options, err := buildParamsFromOperation(op)
	if err != nil {
		t.Errorf("Expected this to work! %s", err)
	}

	if input := op.Params["width"].(int); options.Width != 200 {
		t.Errorf("Expected the Width to be coerced with the correct value of %d", input)
	}

	if input := op.Params["opacity"].(float64); math.Abs(input-float64(options.Opacity)) > epsilon {
		t.Errorf("Expected the Opacity to be coerced with the correct value of %f", input)
	}

	if options.Force != true || options.StripMetadata != false {
		t.Errorf("Expected boolean parameters to result in their respective value's\n%+v", options)
	}

	if input := op.Params["background"].(string); options.Background[0] != 255 {
		t.Errorf("Expected color parameter to be coerced with the correct value of %s", input)
	}
}

// testCase is a generic type to hold a single test case.
type testCase[T any] struct {
	Input  interface{}
	Expect T
	Err    error
}

// runTests is a generic helper that executes the coercion function against
// all the given test cases. It verifies that errors and values match expectations.
func runTests[T any](t *testing.T, name string, cases []testCase[T],
	coerceFn func(interface{}) (T, error), equal func(a, b T) bool) {
	t.Run(name, func(t *testing.T) {
		for _, tc := range cases {
			result, err := coerceFn(tc.Input)
			if err != nil && tc.Err == nil {
				t.Errorf(DidNotExpectError, err, tc)
				t.FailNow()
			}
			if tc.Err != nil && tc.Err != err {
				t.Errorf(ExpectedError, tc.Err, err)
				t.FailNow()
			}
			if tc.Err == nil && !equal(result, tc.Expect) {
				t.Errorf(ExpectedProperCoercion, err, result, tc)
			}
		}
	})
}

func TestCoerceTypeFns(t *testing.T) {
	runTests[int](t, "coerceTypeInt", []testCase[int]{
		{Input: "200", Expect: 200},
		{Input: int(200), Expect: 200},
		{Input: float64(200), Expect: 200},
		{Input: false, Expect: 0, Err: ErrUnsupportedValue},
	}, coerceTypeInt, func(a, b int) bool { return a == b })

	runTests[float64](t, "coerceTypeFloat", []testCase[float64]{
		{Input: "200", Expect: 200},
		{Input: int(200), Expect: 200},
		{Input: float64(200), Expect: 200},
		{Input: false, Expect: 0, Err: ErrUnsupportedValue},
	}, coerceTypeFloat, func(a, b float64) bool {
		return math.Abs(a-b) <= epsilon
	})

	runTests[bool](t, "coerceTypeBool", []testCase[bool]{
		{Input: "true", Expect: true},
		{Input: true, Expect: true},
		{Input: "1", Expect: true},
		{Input: "bubblegum", Expect: false, Err: ErrUnsupportedValue},
	}, coerceTypeBool, func(a, b bool) bool { return a == b })

	runTests[string](t, "coerceTypeString", []testCase[string]{
		{Input: "true", Expect: "true"},
		{Input: false, Err: ErrUnsupportedValue},
		{Input: 0.0, Err: ErrUnsupportedValue},
		{Input: 0, Err: ErrUnsupportedValue},
	}, coerceTypeString, func(a, b string) bool { return a == b })
}
