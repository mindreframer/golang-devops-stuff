package js

import (
	"github.com/mailgun/vulcan/netutils"
	"net/http"
)

// Converts http request to json object that will be visible in the javascript handler
func requestToJs(r *http.Request) (map[string]interface{}, error) {
	auth, err := netutils.ParseAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		auth = &netutils.BasicAuth{}
	}

	return map[string]interface{}{
		// Note that the auth property and it's members exist
		// regardless of the fact if header was supplied at all
		// to simplify logic in the javascript handler
		"auth": map[string]interface{}{
			"username": auth.Username,
			"password": auth.Password,
		},
		"url":      r.URL.String(),
		"query":    r.URL.Query(),
		"path":     r.URL.Path,
		"protocol": r.Proto,
		"method":   r.Method,
		"length":   r.ContentLength,
		"headers":  r.Header,
	}, nil
}
