package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	parameterWidth    = "w"
	parameterHeight   = "h"
	parameterCropping = "c"
	parameterGravity  = "g"
	parameterFilter   = "f"
	parameterScale    = "s"

	// CroppingModeExact crops an image exactly to given dimensions
	CroppingModeExact = "e"
	// CroppingModeAll crops an image so that all of it is displayed in a frame of at most given dimensions
	CroppingModeAll = "a"
	// CroppingModePart crops an image so that it fills a frame of given dimensions
	CroppingModePart = "p"
	// CroppingModeKeepScale crops an image so that it fills a frame of given dimensions, keeps scale
	CroppingModeKeepScale = "k"

	GravityNorth     = "n"
	GravityNorthEast = "ne"
	GravityEast      = "e"
	GravitySouthEast = "se"
	GravitySouth     = "s"
	GravitySouthWest = "sw"
	GravityWest      = "w"
	GravityNorthWest = "nw"
	GravityCenter    = "c"

	FilterGrayScale = "grayscale"

	DefaultScale        = 1
	DefaultCroppingMode = CroppingModeExact
	DefaultGravity      = GravityNorthWest
	DefaultFilter       = "none"
)

var (
	transformationNameRe = regexp.MustCompile("^t_([0-9A-Za-z-]+)$")
)

// Params is a struct of parameters specifying an image transformation
type Params struct {
	width, height, scale      int
	cropping, gravity, filter string
}

// ToString turns parameters into a unique string for each possible assignment of parameters
func (p Params) ToString() string {
	// 0 as a value for width or height means that it will be calculated
	return fmt.Sprintf("%s_%s,%s_%s,%s_%d,%s_%d,%s_%s,%s_%d", parameterCropping, p.cropping, parameterGravity, p.gravity, parameterHeight, p.height, parameterWidth, p.width, parameterFilter, p.filter, parameterScale, p.scale)
}

// WithScale returns a copy of a Params struct with the scale set to the given value
func (p Params) WithScale(scale int) Params {
	return Params{p.width, p.height, scale, p.cropping, p.gravity, p.filter}
}

// Turns a string like "w_400,h_300" and an image path into a Params struct
// The second return value is an error message
// Also validates the parameters to make sure they have valid values
// w = width, h = height
func parseParameters(parametersStr string) (Params, error) {
	params := Params{0, 0, DefaultScale, DefaultCroppingMode, DefaultGravity, DefaultFilter}
	parts := strings.Split(parametersStr, ",")
	for _, part := range parts {
		keyAndValue := strings.SplitN(part, "_", 2)
		key := keyAndValue[0]
		value := keyAndValue[1]

		switch key {
		case parameterWidth, parameterHeight:
			value, err := strconv.Atoi(value)
			if err != nil {
				return params, fmt.Errorf("could not parse value for parameter: %q", key)
			}
			if value <= 0 {
				return params, fmt.Errorf("value %d must be > 0: %q", value, key)
			}
			if key == parameterWidth {
				params.width = value
			} else {
				params.height = value
			}
		case parameterCropping:
			value = strings.ToLower(value)
			if len(value) > 1 {
				return params, fmt.Errorf("value %q must have only 1 character", key)
			}
			if !isValidCroppingMode(value) {
				return params, fmt.Errorf("invalid value for %q", key)
			}
			params.cropping = value
		case parameterGravity:
			value = strings.ToLower(value)
			if len(value) > 2 {
				return params, fmt.Errorf("value %q must have at most 2 characters", key)
			}
			if !isValidGravity(value) {
				return params, fmt.Errorf("invalid value for %q", key)
			}
			params.gravity = value
		case parameterFilter:
			value = strings.ToLower(value)
			if !isValidFilter(value) {
				return params, fmt.Errorf("invalid value for %q", key)
			}
			params.filter = value
		}
	}

	if params.width == 0 && params.height == 0 {
		return params, fmt.Errorf("both width and height can't be 0")
	}

	return params, nil
}

// Parses transformation name from a parameters string (e.g. photo from t_photo).
// Returns "" if there is no transformation name.
func parseTransformationName(parametersStr string) string {
	matches := transformationNameRe.FindStringSubmatch(parametersStr)
	if len(matches) == 0 {
		return ""
	}
	return matches[1]
}

func isValidCroppingMode(str string) bool {
	return str == CroppingModeExact || str == CroppingModeAll || str == CroppingModePart || str == CroppingModeKeepScale
}

func isValidGravity(str string) bool {
	return str == GravityNorth || str == GravityNorthEast || str == GravityEast || str == GravitySouthEast || str == GravitySouth || str == GravitySouthWest || str == GravityWest || str == GravityNorthWest || str == GravityCenter
}

func isValidFilter(str string) bool {
	return str == FilterGrayScale
}

func isEasternGravity(str string) bool {
	return str == GravityNorthEast || str == GravityEast || str == GravitySouthEast
}

func isSouthernGravity(str string) bool {
	return str == GravitySouthWest || str == GravitySouth || str == GravitySouthEast
}
