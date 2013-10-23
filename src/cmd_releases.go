package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	//"strconv"
	"time"

	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

type Release struct {
	Version  string
	Revision string
	Date     time.Time
	Config   map[string]string
}

var (
	awsAuth aws.Auth = aws.Auth{awsKey, awsSecret}
)

func getS3Bucket() *s3.Bucket {
	return s3.New(awsAuth, awsRegion).Bucket(s3BucketName)
}

/*func getNextReleaseVersion(applicationName string) (string, error) {
	releases, err := getReleases(applicationName)
	if err != nil {
		return "", err
	}
	version := "v1"
	if len(releases) > 0 {
		v, err := strconv.Atoi(releases[0].Version[1:])
		if err != nil {
			return version, err
		}
		version = fmt.Sprint("v", v+1)
	}
	fmt.Printf("Next version for %v will be %v", applicationName, version)
	return version, nil
}*/

func getReleases(applicationName string) ([]Release, error) {
	var releases []Release
	bs, err := getS3Bucket().Get(
		"/releases/" + applicationName + "/manifest.json",
	)
	if err != nil {
		if err.Error() == "The specified key does not exist." {
			// The manifest.json file for this app was missing, fill in an empty releases list and continue on our way.
			err = setReleases(applicationName, []Release{})
			if err != nil {
				return releases, err
			}
			fmt.Printf("warn: getReleases S3 key was missing for application \"%v\", so an empty releases list was set", applicationName)
			return []Release{}, err
		}
		return releases, err
	}
	err = json.Unmarshal(bs, &releases)
	return releases, err
}
func setReleases(applicationName string, releases []Release) error {
	bs, err := json.Marshal(releases)
	if err != nil {
		return err
	}
	return getS3Bucket().Put(
		"/releases/"+applicationName+"/manifest.json",
		bs,
		"application/json",
		"private",
	)
}
func delReleases(applicationName string, logger io.Writer) error {
	bucket := getS3Bucket()
	keys, err := bucket.List("releases/"+applicationName, "/releases/"+applicationName, "", 999999)
	if err != nil {
		return err
	}
	fmt.Fprint(logger, "Purging application from S3..\n")
	for _, key := range keys.Contents {
		fmt.Fprintf(logger, "    Deleting key %v\n", key.Key)
		bucket.Del(key.Key)
	}
	return nil
}

func (this *Server) Releases_List(conn net.Conn, applicationName string) error {
	releases, err := getReleases(applicationName)
	if err != nil {
		Logf(conn, "%v", err)
		return err
	}
	for _, r := range releases {
		Logf(conn, "%v %v %v\n", r.Version, r.Revision, r.Date)
	}
	return nil
}
func (this *Server) Releases_Info(conn net.Conn, applicationName, version string) error {
	return fmt.Errorf("not implemented")
}
