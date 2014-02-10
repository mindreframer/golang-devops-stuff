package cloudfront

import (
	"crypto/sha1"
	"encoding/base64"
	"github.com/crowdmob/goamz/aws"
	"log"
	"strconv"
	"time"
)

type CloudFront struct {
	auth    aws.Auth
	BaseURL string
}

func New(auth aws.Auth, baseurl string) *CloudFront {
	return &CloudFront{auth: auth, BaseURL: baseurl}
}

func (cloudfront *CloudFront) SignedURL(path, querystrings string, expires time.Time) string {
	policy := `{"Statement":[{"Resource":"` + cloudfront.BaseURL + "?" + querystrings + `,"Condition":{"DateLessThan":{"AWS:EpochTime":` + strconv.FormatInt(expires.Unix(), 10) + `}}}]}`

	log.Printf("Policy: %v\n", policy)
	hash := sha1.New()
	b := hash.Sum([]byte(policy))
	he := base64.StdEncoding.EncodeToString(b)

	policySha1 := he
	keypairid_str := cloudfront.auth.AccessKey

	url := cloudfront.BaseURL + path + "?" + querystrings + "&Expires=" + strconv.FormatInt(expires.Unix(), 10) + "&Signature=" + policySha1 + "&Key-Pair-Id=" + keypairid_str
	return url
}
