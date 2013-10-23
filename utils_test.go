package errplane

import (
	. "launchpad.net/gocheck"
)

func containsString(slice []string, str string) bool {
	for _, sliceStr := range slice {
		if sliceStr == str {
			return true
		}
	}
	return false
}

type containsStringChecker struct {
	*CheckerInfo
}

func (c *containsStringChecker) Check(params []interface{}, names []string) (result bool, error string) {
	var (
		ok    bool
		slice []string
		value string
	)
	slice, ok = params[0].([]string)
	if !ok {
		return false, "First parameter is not a []int"
	}
	value, ok = params[1].(string)
	if !ok {
		return false, "Second parameter is not an int"
	}
	return containsString(slice, value), ""
}

var Contains Checker = &containsStringChecker{&CheckerInfo{Name: "Contains", Params: []string{"Container", "Expected to contain"}}}
