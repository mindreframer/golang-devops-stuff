package main

// TODO - refactor out a resizing function

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"strconv"
	"strings"

	"crypto/sha1"

	"code.google.com/p/freetype-go/freetype"
	"code.google.com/p/freetype-go/freetype/truetype"
	"github.com/nfnt/resize"
)

// Transformation specifies parameters and a watermark to be used when transforming an image
type Transformation struct {
	params    *Params
	watermark *Watermark
	texts     []*Text
}

// Watermark specifies a watermark to be applied to an image
type Watermark struct {
	imagePath, gravity string
	x, y               int
}

// Text specifies a text overlay to be applied to an image
type Text struct {
	content, gravity, fontFilePath string
	x, y, size                     int
	font                           *truetype.Font
	color                          color.Color
}

// FontMetrics defines font metrics for a Text struct as rounded up integers
type FontMetrics struct {
	width, height, ascent, descent float64
}

// Turns an image file path and a transformation parameters into a file path combining both.
// It can then be used for file lookups.
// The function assumes that imagePath contains an extension at the end.
func (t *Transformation) createFilePath(imagePath string) (string, error) {
	i := strings.LastIndex(imagePath, ".")
	if i == -1 {
		return "", fmt.Errorf("invalid image path")
	}

	sum := make([]byte, sha1.Size)

	// Watermark
	if t.watermark != nil {
		hash := t.watermark.hash()
		for i := range sum {
			sum[i] += hash[i]
		}
	}

	// Texts
	for _, elem := range t.texts {
		hash := elem.hash()
		for i := range sum {
			sum[i] += hash[i]
		}
	}

	extraHash := ""
	if t.watermark != nil || len(t.texts) != 0 {
		extraHash = "--" + hex.EncodeToString(sum)
	}

	return imagePath[:i] + "--" + t.params.ToString() + extraHash + "--" + imagePath[i:], nil
}

func (w *Watermark) hash() []byte {
	h := sha1.New()

	io.WriteString(h, w.imagePath)
	io.WriteString(h, w.gravity)
	io.WriteString(h, strconv.Itoa(w.x))
	io.WriteString(h, strconv.Itoa(w.y))

	return h.Sum(nil)
}

func (t *Text) hash() []byte {
	h := sha1.New()
	writeUint := func(i uint32) {
		bs := make([]byte, 4)
		binary.BigEndian.PutUint32(bs, i)
		h.Write(bs)
	}

	r, g, b, a := t.color.RGBA()
	io.WriteString(h, t.content)
	io.WriteString(h, t.gravity)
	io.WriteString(h, strconv.Itoa(t.x))
	io.WriteString(h, strconv.Itoa(t.y))
	io.WriteString(h, strconv.Itoa(t.size))
	io.WriteString(h, t.fontFilePath)
	writeUint(r)
	writeUint(g)
	writeUint(b)
	writeUint(a)

	return h.Sum(nil)
}

func (t *Text) getFontMetrics(scale int) FontMetrics {
	// Adapted from: https://code.google.com/p/plotinum/

	// Converts truetype.FUnit to float64
	fUnit2Float64 := float64(t.size) / float64(t.font.FUnitsPerEm())

	width := 0
	prev, hasPrev := truetype.Index(0), false
	for _, rune := range t.content {
		index := t.font.Index(rune)
		if hasPrev {
			width += int(t.font.Kerning(t.font.FUnitsPerEm(), prev, index))
		}
		width += int(t.font.HMetric(t.font.FUnitsPerEm(), index).AdvanceWidth)
		prev, hasPrev = index, true
	}
	widthFloat := float64(width) * fUnit2Float64 * float64(scale)

	bounds := t.font.Bounds(t.font.FUnitsPerEm())
	height := float64(bounds.YMax-bounds.YMin) * fUnit2Float64 * float64(scale)
	ascent := float64(bounds.YMax) * fUnit2Float64 * float64(scale)
	descent := float64(bounds.YMin) * fUnit2Float64 * float64(scale)

	return FontMetrics{widthFloat, height, ascent, descent}
}

func transformCropAndResize(img image.Image, transformation *Transformation) (imgNew image.Image) {
	parameters := transformation.params
	width := parameters.width
	height := parameters.height
	gravity := parameters.gravity
	scale := parameters.scale

	imgWidth := img.Bounds().Dx()
	imgHeight := img.Bounds().Dy()

	// Scaling factor
	if parameters.cropping != CroppingModeKeepScale {
		width *= scale
		height *= scale
	}

	// Resize and crop
	switch parameters.cropping {
	case CroppingModeExact:
		imgNew = resize.Resize(uint(width), uint(height), img, resize.Bilinear)
	case CroppingModeAll:
		if float32(width)*(float32(imgHeight)/float32(imgWidth)) > float32(height) {
			// Keep height
			imgNew = resize.Resize(0, uint(height), img, resize.Bilinear)
		} else {
			// Keep width
			imgNew = resize.Resize(uint(width), 0, img, resize.Bilinear)
		}
	case CroppingModePart:
		var croppedRect image.Rectangle
		if float32(width)*(float32(imgHeight)/float32(imgWidth)) > float32(height) {
			// Whole width displayed
			newHeight := int((float32(imgWidth) / float32(width)) * float32(height))
			croppedRect = image.Rect(0, 0, imgWidth, newHeight)
		} else {
			// Whole height displayed
			newWidth := int((float32(imgHeight) / float32(height)) * float32(width))
			croppedRect = image.Rect(0, 0, newWidth, imgHeight)
		}

		topLeftPoint := calculateTopLeftPointFromGravity(gravity, croppedRect.Dx(), croppedRect.Dy(), imgWidth, imgHeight)
		imgDraw := image.NewRGBA(croppedRect)

		draw.Draw(imgDraw, croppedRect, img, topLeftPoint, draw.Src)
		imgNew = resize.Resize(uint(width), uint(height), imgDraw, resize.Bilinear)
	case CroppingModeKeepScale:
		// If passed in dimensions are bigger use those of the image
		if width > imgWidth {
			width = imgWidth
		}
		if height > imgHeight {
			height = imgHeight
		}

		croppedRect := image.Rect(0, 0, width, height)
		topLeftPoint := calculateTopLeftPointFromGravity(gravity, width, height, imgWidth, imgHeight)
		imgDraw := image.NewRGBA(croppedRect)

		draw.Draw(imgDraw, croppedRect, img, topLeftPoint, draw.Src)
		imgNew = imgDraw.SubImage(croppedRect)
	}

	// Filters
	if parameters.filter == FilterGrayScale {
		bounds := imgNew.Bounds()
		w, h := bounds.Max.X, bounds.Max.Y
		gray := image.NewGray(bounds)
		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				oldColor := imgNew.At(x, y)
				grayColor := color.GrayModel.Convert(oldColor)
				gray.Set(x, y, grayColor)
			}
		}
		imgNew = gray
	}

	if transformation.watermark != nil {
		w := transformation.watermark

		var watermarkSrcScaled image.Image
		var watermarkBounds image.Rectangle

		// Try to load a scaled watermark first
		if scale > 1 {
			scaledPath, err := constructScaledPath(w.imagePath, scale)
			if err != nil {
				log.Println("Error:", err)
				return
			}

			watermarkSrc, _, err := loadImage(scaledPath)
			if err != nil {
				log.Println("Error: could not load a watermark", err)
			} else {
				watermarkBounds = watermarkSrc.Bounds()
				watermarkSrcScaled = watermarkSrc
			}
		}

		if watermarkSrcScaled == nil {
			watermarkSrc, _, err := loadImage(w.imagePath)
			if err != nil {
				log.Println("Error: could not load a watermark", err)
				return
			}
			watermarkBounds = image.Rect(0, 0, watermarkSrc.Bounds().Max.X*scale, watermarkSrc.Bounds().Max.Y*scale)
			watermarkSrcScaled = resize.Resize(uint(watermarkBounds.Max.X), uint(watermarkBounds.Max.Y), watermarkSrc, resize.Bilinear)
		}

		bounds := imgNew.Bounds()

		// Make sure we have a transparent watermark if possible
		watermark := image.NewRGBA(watermarkBounds)
		draw.Draw(watermark, watermarkBounds, watermarkSrcScaled, watermarkBounds.Min, draw.Src)

		pt := calculateTopLeftPointFromGravity(w.gravity, watermarkBounds.Dx(), watermarkBounds.Dy(), bounds.Dx(), bounds.Dy())
		pt = pt.Add(getTranslation(w.gravity, w.x*scale, w.y*scale))
		wX := pt.X
		wY := pt.Y

		watermarkRect := image.Rect(wX, wY, watermarkBounds.Dx()+wX, watermarkBounds.Dy()+wY)
		finalImage := image.NewRGBA(bounds)
		draw.Draw(finalImage, bounds, imgNew, bounds.Min, draw.Src)
		draw.Draw(finalImage, watermarkRect, watermark, watermarkBounds.Min, draw.Over)
		imgNew = finalImage.SubImage(bounds)
	}

	if transformation.texts != nil {
		bounds := imgNew.Bounds()
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, imgNew, image.ZP, draw.Src)

		dpi := float64(72) // Multiply this by scale for a baaad time

		c := freetype.NewContext()
		c.SetDPI(dpi)
		c.SetClip(rgba.Bounds())
		c.SetDst(rgba)

		for _, text := range transformation.texts {
			size := float64(text.size * scale)

			c.SetSrc(image.NewUniform(text.color))
			c.SetFont(text.font)
			c.SetFontSize(size)

			fontMetrics := text.getFontMetrics(scale)
			width := int(c.PointToFix32(fontMetrics.width) >> 8)
			height := int(c.PointToFix32(fontMetrics.height) >> 8)

			pt := calculateTopLeftPointFromGravity(text.gravity, width, height, bounds.Dx(), bounds.Dy())
			pt = pt.Add(getTranslation(text.gravity, text.x*scale, text.y*scale))
			x := pt.X
			y := pt.Y + int(c.PointToFix32(fontMetrics.ascent)>>8)

			_, err := c.DrawString(text.content, freetype.Pt(x, y))
			if err != nil {
				log.Println("Error adding text:", err)
				return
			}
		}

		imgNew = rgba
	}

	return
}

func calculateTopLeftPointFromGravity(gravity string, width, height, imgWidth, imgHeight int) image.Point {
	// Assuming width <= imgWidth && height <= imgHeight
	switch gravity {
	case GravityNorth:
		return image.Point{(imgWidth - width) / 2, 0}
	case GravityNorthEast:
		return image.Point{imgWidth - width, 0}
	case GravityEast:
		return image.Point{imgWidth - width, (imgHeight - height) / 2}
	case GravitySouthEast:
		return image.Point{imgWidth - width, imgHeight - height}
	case GravitySouth:
		return image.Point{(imgWidth - width) / 2, imgHeight - height}
	case GravitySouthWest:
		return image.Point{0, imgHeight - height}
	case GravityWest:
		return image.Point{0, (imgHeight - height) / 2}
	case GravityNorthWest:
		return image.Point{0, 0}
	case GravityCenter:
		return image.Point{(imgWidth - width) / 2, (imgHeight - height) / 2}
	}
	panic("This point should not be reached")
}

// getTranslation returns a point specifying a translation by a given
// horizontal and vertical offset according to gravity
func getTranslation(gravity string, h, v int) image.Point {
	switch gravity {
	case GravityNorth:
		return image.Point{0, v}
	case GravityNorthEast:
		return image.Point{-h, v}
	case GravityEast:
		return image.Point{-h, 0}
	case GravitySouthEast:
		return image.Point{-h, -v}
	case GravitySouth:
		return image.Point{0, -v}
	case GravitySouthWest:
		return image.Point{h, -v}
	case GravityWest:
		return image.Point{h, 0}
	case GravityNorthWest:
		return image.Point{h, v}
	case GravityCenter:
		return image.Point{0, 0}
	}
	panic("This point should not be reached")
}
