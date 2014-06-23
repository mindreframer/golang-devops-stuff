package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"code.google.com/p/freetype-go/freetype"

	"github.com/ReshNesh/go-colorful"
	"gopkg.in/yaml.v1"
)

const (
	// LRU = Least recently used
	LRU = "LRU"
	// LFU = Least frequently used
	LFU = "LFU"
)

const (
	defaultThrottlingRate             = 60 // Requests per min
	defaultCacheLimit                 = 0  // No. of bytes
	defaultJpegQuality                = 75
	defaultUploadMaxFileSize          = 5 * 1024 * 1024 // No. of bytes
	defaultUploadMaxPixels            = 5000000         // 5 megapixels
	defaultAllowCustomTransformations = true
	defaultAllowCustomScale           = true
	defaultAsyncUploads               = false
	defaultAuthorisedGet              = false
	defaultAuthorisedUpload           = false
	defaultLocalPath                  = "local-images"
	defaultCacheStrategy              = LRU
	defaultFontPath                   = "fonts/DejaVuSans.ttf"
)

var (
	// Config is a global configuration object
	Config Configuration
)

// Configuration specifies server configuration options
type Configuration struct {
	throttlingRate, cacheLimit, jpegQuality, uploadMaxFileSize, uploadMaxPixels                 int
	allowCustomTransformations, allowCustomScale, asyncUploads, authorisedGet, authorisedUpload bool
	localPath, cacheStrategy                                                                    string
	corsAllowOrigins                                                                            []string
	transformations                                                                             map[string]Transformation
	eagerTransformations                                                                        []Transformation
}

func configInit(configFilePath string) error {
	Config = Configuration{defaultThrottlingRate, defaultCacheLimit, defaultJpegQuality, defaultUploadMaxFileSize, defaultUploadMaxPixels, defaultAllowCustomTransformations, defaultAllowCustomScale, defaultAsyncUploads, defaultAuthorisedGet, defaultAuthorisedUpload, defaultLocalPath, defaultCacheStrategy, nil, make(map[string]Transformation), make([]Transformation, 0)}

	if configFilePath == "" {
		return nil
	}

	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return err
	}

	m := make(map[interface{}]interface{})
	err = yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		return err
	}

	throttlingRate, ok := m["throttling-rate"].(int)
	if ok && throttlingRate >= 0 {
		Config.throttlingRate = throttlingRate
	}

	jpegQuality, ok := m["jpeg-quality"].(int)
	if ok && jpegQuality >= 1 && jpegQuality <= 100 {
		Config.jpegQuality = jpegQuality
	}

	uploadMaxFileSize, ok := m["upload-max-file-size"].(int)
	if ok && uploadMaxFileSize > 0 {
		Config.uploadMaxFileSize = uploadMaxFileSize
	}

	uploadMaxPixels, ok := m["upload-max-pixels"].(int)
	if ok && uploadMaxPixels > 0 {
		Config.uploadMaxPixels = uploadMaxPixels
	}

	allowCustomTransformations, ok := m["allow-custom-transformations"].(bool)
	if ok {
		Config.allowCustomTransformations = allowCustomTransformations
	}

	allowCustomScale, ok := m["allow-custom-scale"].(bool)
	if ok {
		Config.allowCustomScale = allowCustomScale
	}

	asyncUploads, ok := m["async-uploads"].(bool)
	if ok {
		Config.asyncUploads = asyncUploads
	}

	authorisation, ok := m["authorisation"].(map[interface{}]interface{})
	if ok {
		get, ok := authorisation["get"].(bool)
		if ok {
			Config.authorisedGet = get
		}
		upload, ok := authorisation["upload"].(bool)
		if ok {
			Config.authorisedUpload = upload
		}
	}

	localPath, ok := m["local-path"].(string)
	if ok {
		Config.localPath = localPath
	}

	cache, ok := m["cache"].(map[interface{}]interface{})
	if ok {
		limit, ok := cache["limit"].(int)
		if ok {
			Config.cacheLimit = limit
		}

		strategy, ok := cache["strategy"].(string)
		if ok && (strategy == LRU || strategy == LFU) {
			Config.cacheStrategy = strategy
		}
	}

	corsAllowOrigins, ok := m["cors-allow-origins"].([]interface{})
	if ok {
		allowOrigins := make([]string, 0)
		for _, origin := range corsAllowOrigins {
			originStr, ok := origin.(string)
			if ok {
				allowOrigins = append(allowOrigins, originStr)
			}
		}
		Config.corsAllowOrigins = allowOrigins
	}

	transformations, ok := m["transformations"].([]interface{})
	if !ok {
		return nil
	}

	for _, transformationMap := range transformations {
		transformation, ok := transformationMap.(map[interface{}]interface{})
		if !ok {
			continue
		}

		parametersStr, ok := transformation["parameters"].(string)
		if !ok {
			continue
		}

		params, err := parseParameters(parametersStr)
		if err != nil {
			return fmt.Errorf("invalid transformation parameters: %s (%s)", parametersStr, err)
		}

		name, ok := transformation["name"].(string)
		if !ok {
			continue
		}
		if !isValidTransformationName(name) {
			return fmt.Errorf("invalid transformation name: %s", name)
		}

		t := Transformation{&params, nil, make([]*Text, 0)}

		watermarkMap, ok := transformation["watermark"].(map[interface{}]interface{})
		if ok {
			imagePath, ok := watermarkMap["source"].(string)
			if !ok {
				return fmt.Errorf("a watermark needs to have a source specified")
			}

			gravity, ok := watermarkMap["gravity"].(string)
			if !ok || !isValidGravity(gravity) {
				return fmt.Errorf("missing or invalid gravity: %s", gravity)
			}

			// x and y will default to 0 if not found in config
			x, ok := watermarkMap["x-pos"].(int)
			if x < 0 {
				return fmt.Errorf("x-pos must be at least 0")
			}
			y, ok := watermarkMap["y-pos"].(int)
			if y < 0 {
				return fmt.Errorf("y-pos must be at least 0")
			}

			t.watermark = &Watermark{imagePath, gravity, x, y}
		}

		texts, ok := transformation["text"].([]interface{})
		if ok {

			for _, textMap := range texts {
				text, ok := textMap.(map[interface{}]interface{})
				if !ok {
					continue
				}

				content, ok := text["content"].(string)

				gravity, ok := text["gravity"].(string)
				if !ok || !isValidGravity(gravity) {
					return fmt.Errorf("missing or invalid gravity: %s", gravity)
				}

				// x and y will default to 0 if not found in config
				x, ok := text["x-pos"].(int)
				if x < 0 {
					return fmt.Errorf("x-pos must be at least 0")
				}
				y, ok := text["y-pos"].(int)
				if y < 0 {
					return fmt.Errorf("y-pos must be at least 0")
				}

				colorStr, ok := text["color"].(string)
				if !ok {
					return fmt.Errorf("text needs to have a color specified")
				}
				color, err := colorful.Hex(colorStr)
				if err != nil {
					return err
				}

				fontFilePath, ok := text["font"].(string)
				if !ok {
					fontFilePath = defaultFontPath
				}
				if _, err := os.Stat(fontFilePath); os.IsNotExist(err) {
					return fmt.Errorf("font does not exist: %s", fontFilePath)
				}
				fontBytes, err := ioutil.ReadFile(fontFilePath)
				if err != nil {
					return fmt.Errorf("loading font failed: %s", err)
				}
				font, err := freetype.ParseFont(fontBytes)
				if err != nil {
					return fmt.Errorf("loading font failed: %s", err)
				}

				size, ok := text["size"].(int)
				if !ok {
					return fmt.Errorf("%v is not a valid size %s", text["size"])
				}
				if size < 1 {
					return fmt.Errorf("size needs to be at least 1")
				}

				t.texts = append(t.texts, &Text{content, gravity, fontFilePath, x, y, size, font, color})
			}
		}

		Config.transformations[name] = t

		eager, ok := transformation["eager"].(bool)
		if ok && eager {
			Config.eagerTransformations = append(Config.eagerTransformations, t)
		}
	}

	return nil
}

var (
	transformationNameConfigRe = regexp.MustCompile("^([0-9A-Za-z-]+)$")
)

func isValidTransformationName(name string) bool {
	return transformationNameConfigRe.MatchString(name)
}
