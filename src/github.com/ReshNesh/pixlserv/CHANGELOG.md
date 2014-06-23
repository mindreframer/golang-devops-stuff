## 0.4

Features:

- a configuration option for the maximum number of pixels in an image as a prevention of PNG decompression bombs

Bug fixes:

- nicer error message when connecting to redis fails

## 0.3

Features:

- uploads need to be signed using a secret key (when authentication is configured to be required for uploads)

## 0.2

Features:

- text overlays
- watermarks and text overlays positioning using gravity
- Google Cloud Storage support (by @hjr265)

Bug fixes:

- generated file name clash when transformation parameters other than watermark and text overlays are the same
