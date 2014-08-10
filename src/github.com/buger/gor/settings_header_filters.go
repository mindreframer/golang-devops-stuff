package main

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type headerFilter struct {
	name   string
	regexp *regexp.Regexp
}

type HTTPHeaderFilters []headerFilter

func (h *HTTPHeaderFilters) String() string {
	return fmt.Sprint(*h)
}

func (h *HTTPHeaderFilters) Set(value string) error {
	valArr := strings.SplitN(value, ":", 2)
	if len(valArr) < 2 {
		return errors.New("need both header and value, colon-delimited (ex. user_id:^169$).")
	}
	r, err := regexp.Compile(valArr[1])
	if err != nil {
		return err
	}

	*h = append(*h, headerFilter{name: valArr[0], regexp: r})

	return nil
}

func (h *HTTPHeaderFilters) Good(req *http.Request) bool {
	for _, f := range *h {
		if !f.regexp.MatchString(req.Header.Get(f.name)) {
			return false
		}
	}
	return true
}
