package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"code.google.com/p/goauth2/oauth/jwt"
	gcs "code.google.com/p/google-api-go-client/storage/v1beta1"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

const (
	awsKeyEnvVar    = "AWS_ACCESS_KEY_ID"
	awsSecretEnvVar = "AWS_SECRET_ACCESS_KEY"
	s3BucketEnvVar  = "PIXLSERV_S3_BUCKET"

	gcsIssEnvVar    = "GCS_ISS"
	gcsKeyEnvVar    = "GCS_KEY"
	gcsBucketEnvVar = "PIXLSERV_GCS_BUCKET"
)

var (
	storageImpl storage
)

type storage interface {
	init() error

	loadImage(imagePath string) (image.Image, string, error)

	saveImage(img image.Image, format string, imagePath string) (int, error)

	deleteImage(imagePath string) error

	imageExists(imagePath string) bool
}

func storageInit() error {
	if os.Getenv(awsKeyEnvVar) != "" && os.Getenv(awsSecretEnvVar) != "" && os.Getenv(s3BucketEnvVar) != "" {
		storageImpl = new(s3Storage)
		log.Println("Using S3 storage")
	} else if os.Getenv(gcsIssEnvVar) != "" && os.Getenv(gcsKeyEnvVar) != "" && os.Getenv(gcsBucketEnvVar) != "" {
		storageImpl = new(gcsStorage)
		log.Println("Using GCS storage")
	} else {
		storageImpl = new(localStorage)
		log.Println("Using local storage")
	}

	return storageImpl.init()
}

func storageCleanUp() {
}

func loadImage(imagePath string) (image.Image, string, error) {
	return storageImpl.loadImage(imagePath)
}

func saveImage(img image.Image, format string, imagePath string) (int, error) {
	return storageImpl.saveImage(img, format, imagePath)
}

func deleteImage(imagePath string) error {
	return storageImpl.deleteImage(imagePath)
}

func imageExists(imagePath string) bool {
	return storageImpl.imageExists(imagePath)
}

// localStorage is a storage implementation using local disk
type localStorage struct {
	path string
}

func (s *localStorage) init() error {
	path := Config.localPath
	s.path = path
	return nil
}

func (s *localStorage) loadImage(imagePath string) (image.Image, string, error) {
	reader, err := os.Open(s.path + "/" + imagePath)
	defer reader.Close()

	if err != nil {
		return nil, "", fmt.Errorf("image not found: %q", imagePath)
	}
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, "", fmt.Errorf("cannot decode image: %q", imagePath)
	}
	return img, format, nil
}

func (s *localStorage) saveImage(img image.Image, format string, imagePath string) (int, error) {
	// Open file for writing, overwrite if it already exists
	fullPath := s.path + "/" + imagePath
	writer, err := os.Create(fullPath)
	defer writer.Close()

	if err != nil {
		return 0, err
	}

	err = writeImage(img, format, writer)
	if err != nil {
		return 0, err
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return 0, nil
	}

	stat, err := f.Stat()
	if err != nil {
		return 0, nil
	}
	size := stat.Size()

	return int(size), err
}

func (s *localStorage) deleteImage(imagePath string) error {
	return os.Remove(s.path + "/" + imagePath)
}

func (s *localStorage) imageExists(imagePath string) bool {
	if _, err := os.Stat(s.path + "/" + imagePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// s3Storage is a storage implementation using Amazon S3
type s3Storage struct {
	bucket *s3.Bucket
}

func (s *s3Storage) init() error {
	auth, err := aws.EnvAuth()
	if err != nil {
		return err
	}

	bucketName := os.Getenv(s3BucketEnvVar)
	if bucketName == "" {
		return fmt.Errorf("%s not set", s3BucketEnvVar)
	}

	conn := s3.New(auth, aws.EUWest)
	s.bucket = conn.Bucket(bucketName)

	return nil
}

func (s *s3Storage) loadImage(imagePath string) (image.Image, string, error) {
	data, err := s.bucket.Get(imagePath)
	if err != nil {
		return nil, "", err
	}

	format := strings.TrimLeft(filepath.Ext(imagePath), ".")
	image, err := readImage(data, format)
	if err != nil {
		return nil, "", err
	}

	return image, format, nil
}

func (s *s3Storage) saveImage(img image.Image, format string, imagePath string) (int, error) {
	var buffer bytes.Buffer
	err := writeImage(img, format, &buffer)
	if err != nil {
		return 0, err
	}

	size := buffer.Len()
	return size, s.bucket.Put(imagePath, buffer.Bytes(), "image/"+format, s3.Private)
}

func (s *s3Storage) deleteImage(imagePath string) error {
	return s.bucket.Del(imagePath)
}

func (s *s3Storage) imageExists(imagePath string) bool {
	resp, err := s.bucket.List(imagePath, "/", "", 10)
	if err != nil {
		log.Printf("Error while listing S3 bucket: %s\n", err.Error())
		return false
	}
	if resp == nil {
		log.Println("Error while listing S3 bucket: empty response")
	}

	for _, element := range resp.Contents {
		if element.Key == imagePath {
			return true
		}
	}

	return false
}

// gcsStorage is a storage implementation using Google Cloud Storage
type gcsStorage struct {
	client  *http.Client
	service *gcs.Service
	bucket  string
}

func (s *gcsStorage) init() error {
	jwtToken := jwt.NewToken(os.Getenv(gcsIssEnvVar), gcs.DevstorageRead_writeScope, []byte(os.Getenv(gcsKeyEnvVar)))
	oauthToken, err := jwtToken.Assert(http.DefaultClient)
	if err != nil {
		return err
	}

	client := (&jwt.Transport{jwtToken, oauthToken, http.DefaultTransport}).Client()

	service, err := gcs.New(client)
	if err != nil {
		return err
	}

	s.client = client
	s.service = service
	s.bucket = os.Getenv(gcsBucketEnvVar)

	return nil
}

func (s *gcsStorage) loadImage(imagePath string) (image.Image, string, error) {
	obj, err := s.service.Objects.Get(s.bucket, imagePath).Do()
	if err != nil {
		return nil, "", err
	}

	resp, err := s.client.Get(obj.Media.Link)
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, "", err
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, "", err
	}

	format := strings.TrimLeft(filepath.Ext(imagePath), ".")
	image, err := readImage(buf.Bytes(), format)
	if err != nil {
		return nil, "", err
	}

	return image, format, nil
}

func (s *gcsStorage) saveImage(img image.Image, format string, imagePath string) (int, error) {
	buffer := &bytes.Buffer{}
	err := writeImage(img, format, buffer)
	if err != nil {
		return 0, err
	}

	size := buffer.Len()
	_, err = s.service.Objects.Insert(s.bucket, &gcs.Object{Name: imagePath}).Media(buffer).Do()
	if err != nil {
		return 0, err
	}
	return size, err
}

func (s *gcsStorage) deleteImage(imagePath string) error {
	return s.service.Objects.Delete(s.bucket, imagePath).Do()
}

func (s *gcsStorage) imageExists(imagePath string) bool {
	obj, err := s.service.Objects.Get(s.bucket, imagePath).Do()
	if err != nil {
		return false
	}
	return obj != nil
}
