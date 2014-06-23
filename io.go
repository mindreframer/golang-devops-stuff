package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"regexp"
	"strconv"
)

var (
	scaledPathRe    = regexp.MustCompile("(.+)@(\\d+)x\\.([^\\.]+)$")
	notScaledPathRe = regexp.MustCompile("(.+)\\.([^\\.]+)$")
)

// Writes a given image of the given format to the given destination.
// Returns error.
func writeImage(img image.Image, format string, w io.Writer) error {
	if format == "png" {
		return png.Encode(w, img)
	}
	return jpeg.Encode(w, img, &jpeg.Options{Config.jpegQuality})
}

func readImage(data []byte, format string) (image.Image, error) {
	reader := bytes.NewReader(data)
	if format == "png" {
		return png.Decode(reader)
	}
	return jpeg.Decode(reader)
}

// Returns image@2x.jpg if image.jpg, 2 is passed in
func constructScaledPath(path string, scale int) (string, error) {
	matches := notScaledPathRe.FindStringSubmatch(path)
	if len(matches) == 0 {
		return "", fmt.Errorf("can't parse path: %s", path)
	}
	return fmt.Sprintf("%s@%dx.%s", matches[1], scale, matches[2]), nil
}

// Gets (image.jpg, 2) from image@2x.jpg
func parseBasePathAndScale(path string) (string, int) {
	matches := scaledPathRe.FindStringSubmatch(path)
	if len(matches) == 0 {
		return path, 1
	}
	path = matches[1] + "." + matches[3]
	scale, _ := strconv.Atoi(matches[2])
	return path, scale
}
