package main

import (
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"strings"
)

type headerHashFilter struct {
	name    string
	maxHash uint32
}

type HTTPHeaderHashFilters []headerHashFilter

func (h *HTTPHeaderHashFilters) String() string {
	return fmt.Sprint(*h)
}

func (h *HTTPHeaderHashFilters) Set(value string) error {
	valArr := strings.SplitN(value, ":", 2)
	if len(valArr) < 2 {
		return errors.New("need both header and value, colon-delimited (ex. user_id:1/2).")
	}

	fracArr := strings.Split(valArr[1], "/")
	if len(fracArr) < 2 {
		return errors.New("need both a numerator and denominator specified, slash-delimited (ex. user_id:1/4).")
	}

	var num, den uint64
	num, _ = strconv.ParseUint(fracArr[0], 10, 64)
	den, _ = strconv.ParseUint(fracArr[1], 10, 64)

	if num < 1 || den < 1 || num > den {
		panic("need positive numerators and denominators, with the former less than the latter.")
	}

	for test := den; test != 1; test /= 2 {
		if test%2 == 1 {
			return errors.New("must have a denominator which is a power of two.")
		}
	}

	var f headerHashFilter
	f.name = valArr[0]
	f.maxHash = (uint32)(num * (((uint64)(2 << 31)) / den))
	*h = append(*h, f)

	return nil
}

func (h *HTTPHeaderHashFilters) Good(req *http.Request) bool {
	for _, f := range *h {
		if req.Header.Get(f.name) == "" {
			return false
		}
		hasher := fnv.New32a()
		hasher.Write([]byte(req.Header.Get(f.name)))
		if hasher.Sum32() > f.maxHash {
			return false
		}
	}
	return true
}
